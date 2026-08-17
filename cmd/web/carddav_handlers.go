// CardDAV admin handlers: the /god/carddav page for stats and VCF upload/download.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/borumbombum/borum-api/internal/auth"
	"github.com/emersion/go-vcard"
	"github.com/go-chi/chi/v5"
)

// godCardDAVHandler renders the CardDAV admin page with stats and upload/download buttons.
func (a *app) godCardDAVHandler(w http.ResponseWriter, r *http.Request) {
	sess, sessOk := auth.SessionFrom(r.Context())

	var contactCount int
	var addressBookName string
	err := a.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM contacts c
		 JOIN address_books ab ON c.address_book_id = ab.id
		 WHERE ab.user_id = 'default'`,
	).Scan(&contactCount)
	if err != nil {
		a.errorLogger.Printf("carddav stats: %v", err)
	}

	err = a.db.QueryRowContext(r.Context(),
		`SELECT name FROM address_books WHERE user_id = 'default' LIMIT 1`,
	).Scan(&addressBookName)
	if err != nil {
		addressBookName = "Contacts"
	}

	data := struct {
		godData
		Host            string
		ContactCount    int
		AddressBookName string
	}{
		godData:         a.newGodData(r, false),
		Host:            r.Host,
		ContactCount:    contactCount,
		AddressBookName: addressBookName,
	}

	if sessOk {
		data.CSRF = a.auth.CSRF(sess.Token)
	}

	renderGodPage(w, "god_carddav", data)
}

// godCardDAVUploadHandler handles VCF file uploads via AJAX.
func (a *app) godCardDAVUploadHandler(w http.ResponseWriter, r *http.Request) {
	a.carddavMu.Lock()
	if time.Since(a.carddavLastOK) < 10*time.Second {
		a.carddavMu.Unlock()
		jsonError(w, "Upload in progress, try again later", http.StatusTooManyRequests)
		return
	}
	a.carddavMu.Unlock()
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, "Failed to parse upload", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("vcf")
	if err != nil {
		jsonError(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		jsonError(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	decoder := vcard.NewDecoder(strings.NewReader(string(data)))
	seen := make(map[string]bool)
	contactsImported := 0

	for {
		card, err := decoder.Decode()
		if err == io.EOF {
			break
		}
		if err != nil {
			a.errorLogger.Printf("carddav decode: %v", err)
			continue
		}

		uidField := card.Get(vcard.FieldUID)
		var uid string
		if uidField != nil && uidField.Value != "" {
			uid = uidField.Value
		} else {
			// Fallback: hash name+email+phone for dedup key.
			var buf strings.Builder
			if fn := card.Get(vcard.FieldFormattedName); fn != nil {
				buf.WriteString(fn.Value)
			}
			if em := card.Get(vcard.FieldEmail); em != nil {
				buf.WriteString(em.Value)
			}
			if tel := card.Get(vcard.FieldTelephone); tel != nil {
				buf.WriteString(tel.Value)
			}
			h := [32]byte{}
			copy(h[:], buf.String())
			uid = fmt.Sprintf("carddav-%x", h[:8])
		}

		if seen[uid] {
			continue
		}
		seen[uid] = true

		// Re-encode the individual card for storage.
		var cardBuf strings.Builder
		if err := vcard.NewEncoder(&cardBuf).Encode(card); err != nil {
			a.errorLogger.Printf("carddav encode: %v", err)
			continue
		}
		cardData := cardBuf.String()

		_, err = a.db.ExecContext(r.Context(),
			`INSERT INTO contacts (id, address_book_id, path, vcard_data, etag, updated_at, deleted_at)
			 SELECT ?, ab.id, ?, ?, ?, datetime('now'), NULL
			 FROM address_books ab WHERE ab.user_id = 'default'
			 ON CONFLICT(address_book_id, path) DO UPDATE SET
			 vcard_data = excluded.vcard_data,
			 etag = excluded.etag,
			 updated_at = excluded.updated_at,
			 deleted_at = NULL`,
			uid,
			"/"+uid+".vcf",
			cardData,
			fmt.Sprintf(`"%x"`, len(cardData)),
		)
		if err != nil {
			a.errorLogger.Printf("carddav insert: %v", err)
			continue
		}
		contactsImported++
		log.Printf("carddav import: %d contacts imported", contactsImported)
	}

	a.carddavMu.Lock()
	a.carddavLastOK = time.Now()
	a.carddavMu.Unlock()
	jsonOK(w, map[string]any{"imported": contactsImported})
}

// godCardDAVDownloadHandler exports all contacts as a VCF file.
func (a *app) godCardDAVDownloadHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.QueryContext(r.Context(),
		`SELECT c.vcard_data FROM contacts c
		 JOIN address_books ab ON c.address_book_id = ab.id
		 WHERE ab.user_id = 'default'`,
	)
	if err != nil {
		a.errorLogger.Printf("carddav query: %v", err)
		http.Error(w, "Failed to query contacts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/vcard; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"contacts.vcf\"")

	encoder := vcard.NewEncoder(w)
	for rows.Next() {
		var vcardData string
		if err := rows.Scan(&vcardData); err != nil {
			a.errorLogger.Printf("carddav scan: %v", err)
			continue
		}

		decoder := vcard.NewDecoder(strings.NewReader(vcardData))
		card, err := decoder.Decode()
		if err != nil {
			a.errorLogger.Printf("carddav decode: %v", err)
			continue
		}

		if err := encoder.Encode(card); err != nil {
			a.errorLogger.Printf("carddav encode: %v", err)
			continue
		}
	}
}

// godCardDAVPurgeHandler deletes all contacts.
func (a *app) godCardDAVPurgeHandler(w http.ResponseWriter, r *http.Request) {
	result, err := a.db.ExecContext(r.Context(), `DELETE FROM contacts`)
	if err != nil {
		a.errorLogger.Printf("carddav purge: %v", err)
		jsonError(w, "Failed to purge contacts", http.StatusInternalServerError)
		return
	}
	rows, _ := result.RowsAffected()
	jsonOK(w, map[string]any{"purged": rows})
}

// godCardDAVStatsHandler returns contact count as JSON.
func (a *app) godCardDAVStatsHandler(w http.ResponseWriter, r *http.Request) {
	var count int
	err := a.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM contacts c
		 JOIN address_books ab ON c.address_book_id = ab.id
		 WHERE ab.user_id = 'default'`,
	).Scan(&count)
	if err != nil {
		a.errorLogger.Printf("carddav stats: %v", err)
		jsonError(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]any{"contacts": count})
}

// godCardDAVContactsHandler lists all contacts with parsed name/email/phone.
func (a *app) godCardDAVContactsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.QueryContext(r.Context(),
		`SELECT c.id, c.vcard_data FROM contacts c
		 JOIN address_books ab ON c.address_book_id = ab.id
		 WHERE ab.user_id = 'default'`,
	)
	if err != nil {
		a.errorLogger.Printf("carddav contacts query: %v", err)
		http.Error(w, "Failed to query contacts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type contact struct {
		Index int
		ID    string
		Name  string
		Email string
		Phone string
	}
	var contacts []contact
	idx := 0
	for rows.Next() {
		var id, vcardData string
		if err := rows.Scan(&id, &vcardData); err != nil {
			a.errorLogger.Printf("carddav contacts scan: %v", err)
			continue
		}
		idx++
		c := contact{Index: idx, ID: id, Name: id}
		decoder := vcard.NewDecoder(strings.NewReader(vcardData))
		if card, err := decoder.Decode(); err == nil {
			if fn := card.Get(vcard.FieldFormattedName); fn != nil && fn.Value != "" {
				c.Name = fn.Value
			}
			if email := card.Get(vcard.FieldEmail); email != nil && email.Value != "" {
				c.Email = email.Value
			}
			if tel := card.Get(vcard.FieldTelephone); tel != nil && tel.Value != "" {
				c.Phone = tel.Value
			}
		}
		contacts = append(contacts, c)
	}

	data := struct {
		godData
		Contacts []contact
		Total    int
	}{
		godData:  a.newGodData(r, false),
		Contacts: contacts,
		Total:    len(contacts),
	}

	renderGodPage(w, "god_carddav_contacts", data)
}

// godCardDAVContactDeleteHandler deletes a single contact by ID.
func (a *app) godCardDAVContactDeleteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing contact ID", http.StatusBadRequest)
		return
	}
	_, err := a.db.ExecContext(r.Context(),
		`DELETE FROM contacts WHERE id = ?`, id,
	)
	if err != nil {
		a.errorLogger.Printf("carddav delete contact: %v", err)
	}
	http.Redirect(w, r, "/god/carddav/contacts", http.StatusSeeOther)
}

// godCardDAVContactDownloadHandler downloads a single contact's vCard.
func (a *app) godCardDAVContactDownloadHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing contact ID", http.StatusBadRequest)
		return
	}
	var vcardData string
	err := a.db.QueryRowContext(r.Context(),
		`SELECT c.vcard_data FROM contacts c
		 JOIN address_books ab ON c.address_book_id = ab.id
		 WHERE c.id = ? AND ab.user_id = 'default'`, id,
	).Scan(&vcardData)
	if err != nil {
		a.errorLogger.Printf("carddav download contact: %v", err)
		http.Error(w, "Contact not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/vcard; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.vcf\"", id))
	w.Write([]byte(vcardData))
}

// renderGodPage renders an admin page template.
func renderGodPage(w http.ResponseWriter, name string, data any) {
	t, ok := godTemplates[name]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "godbase", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
