/* god-autosave.js — slug generation from title and auto-save drafts for the
   article form. Reads form fields, debounces changes, and saves via the JSON
   API. Shows a Lucide loader icon during saves. Only active on pages with
   [data-autosave] (the article form). */
(function () {
    'use strict';
    var doc = document;
    var win = window;

    var form = doc.querySelector('[data-autosave]');
    if (!form) return;

    var titleInput = form.querySelector('[name="title"]');
    var slugInput = form.querySelector('#slug-input');
    var statusEl = doc.getElementById('autosave-status');
    var DEBOUNCE = 2000;
    var timer = null;
    var currentSlug = slugInput ? slugInput.value : (form.getAttribute('data-slug') || '');
    var hasContent = false;

    /* ------------------------------------------------ slug generation */
    var slugRe = /[^a-z0-9-]/g;
    var dashRe = /-{2,}/g;

    function generateSlug(title) {
        var s = title.toLowerCase().replace(/ /g, '-').replace(slugRe, '').replace(dashRe, '-').replace(/^-|-$/g, '');
        return s || 'untitled';
    }

    // Auto-generate slug from title only when no slug exists yet (new article).
    if (slugInput && titleInput) {
        titleInput.addEventListener('input', function () {
            if (!currentSlug) {
                slugInput.value = generateSlug(titleInput.value);
            }
        });
    }

    /* ------------------------------------------------ status display */
    function setStatus(html) {
        if (!statusEl) return;
        statusEl.innerHTML = html;
        if (win.lucide && win.lucide.createIcons) {
            win.lucide.createIcons();
        }
    }

    var ICON_SPINNER = '<i data-lucide="loader-2" width="14" height="14" class="inline-block animate-spin"></i>';
    var ICON_CHECK = '<i data-lucide="check" width="14" height="14" class="inline-block text-green-600"></i>';
    var ICON_ALERT = '<i data-lucide="alert-circle" width="14" height="14" class="inline-block text-red-600"></i>';

    /* ------------------------------------------------ form data */
    function collectData() {
        var data = {};
        var fields = ['title', 'subtitle', 'date', 'excerpt', 'image', 'image_caption', 'body'];
        fields.forEach(function (name) {
            var el = form.querySelector('[name="' + name + '"]');
            data[name] = el ? el.value : '';
        });
        var tagsEl = form.querySelector('[name="tags"]');
        if (tagsEl) {
            data.tags = tagsEl.value.split(',').map(function (t) {
                return t.trim();
            }).filter(Boolean);
        }
        var starEl = form.querySelector('[name="star"]');
        data.star = starEl ? starEl.checked : false;
        var featuredEl = form.querySelector('[name="featured"]');
        data.featured = featuredEl ? featuredEl.checked : false;
        // Always include slug from the input so the backend knows the current value.
        if (slugInput) {
            data.slug = slugInput.value;
        }
        return data;
    }

    /* ------------------------------------------------ save */
    function save() {
        if (!titleInput || !titleInput.value.trim()) {
            hasContent = false;
            return;
        }
        hasContent = true;
        setStatus(ICON_SPINNER + ' <span class="font-mono text-[11px]">saving…</span>');

        var data = collectData();
        var url, method;
        if (currentSlug) {
            url = '/api/v1/articles/' + encodeURIComponent(currentSlug) + '/draft';
            method = 'PUT';
        } else {
            url = '/api/v1/articles/draft';
            method = 'POST';
        }

        var xhr = new XMLHttpRequest();
        xhr.open(method, url, true);
        xhr.setRequestHeader('Content-Type', 'application/json');
        xhr.onreadystatechange = function () {
            if (xhr.readyState !== 4) return;
            if (xhr.status >= 200 && xhr.status < 300) {
                try {
                    var resp = JSON.parse(xhr.responseText);
                    if (resp.slug) {
                        // Update tracked slug (handles slug changes for drafts).
                        var slugChanged = resp.slug !== currentSlug;
                        currentSlug = resp.slug;
                        if (slugInput) slugInput.value = resp.slug;
                        if (slugChanged || !history.state || !history.state.slug) {
                            history.replaceState({ slug: resp.slug }, '', '/god/articles/' + resp.slug + '/edit');
                        }
                    }
                } catch (e) { /* ignore parse errors */ }
                setStatus(ICON_CHECK + ' <span class="font-mono text-[11px]">saved</span>');
            } else {
                setStatus(ICON_ALERT + ' <span class="font-mono text-[11px]">error</span>');
            }
        };
        xhr.onerror = function () {
            setStatus(ICON_ALERT + ' <span class="font-mono text-[11px]">error</span>');
        };
        xhr.send(JSON.stringify(data));
    }

    function scheduleSave() {
        clearTimeout(timer);
        timer = setTimeout(save, DEBOUNCE);
    }

    /* ------------------------------------------------ listen for changes */
    form.addEventListener('input', scheduleSave);

    // TipTap editor syncs to textarea via god-editor.js onUpdate.
    // Poll the body textarea for changes to catch TipTap updates.
    var bodyEl = form.querySelector('[name="body"]');
    if (bodyEl) {
        var lastBody = bodyEl.value;
        setInterval(function () {
            if (bodyEl.value !== lastBody) {
                lastBody = bodyEl.value;
                scheduleSave();
            }
        }, 1000);
    }

    /* ------------------------------------------------ preview */
    var previewBtn = form.querySelector('[data-preview]');
    if (previewBtn) {
        previewBtn.addEventListener('click', function () {
            var url = previewBtn.getAttribute('data-preview');
            var csrf = form.querySelector('[name="_csrf"]');
            if (!url || !csrf) return;
            var idle = previewBtn.querySelector('[data-preview-idle]');
            var loading = previewBtn.querySelector('[data-preview-loading]');
            if (idle) idle.hidden = true;
            if (loading) loading.hidden = false;
            previewBtn.disabled = true;
            if (window.lucide && window.lucide.createIcons) window.lucide.createIcons();
            var xhr = new XMLHttpRequest();
            xhr.open('POST', url, true);
            xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
            xhr.onreadystatechange = function () {
                if (xhr.readyState === 4) {
                    previewBtn.disabled = false;
                    if (loading) loading.hidden = true;
                    if (idle) idle.hidden = false;
                    if (window.lucide && window.lucide.createIcons) window.lucide.createIcons();
                    if (xhr.status >= 200 && xhr.status < 400) {
                        var finalUrl = xhr.responseURL;
                        if (finalUrl) window.open(finalUrl, '_blank');
                    }
                }
            };
            xhr.send('_csrf=' + encodeURIComponent(csrf.value));
        });
    }

    /* ------------------------------------------------ form submission */
    // The "save" button publishes; the "save draft" button saves as draft.
    // Set the form action based on which button was clicked.
    form.addEventListener('submit', function (e) {
        clearTimeout(timer);
        var btn = e.submitter;
        if (btn && btn.getAttribute('data-action') === 'draft') {
            form.setAttribute('action', '/god/articles/draft');
        }
    });
})();
