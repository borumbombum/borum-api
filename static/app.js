/* DOaGS — static port
 * Client-side behavior only. All page content is baked into the .html files.
 * Mirrors the behavior of the SvelteKit original (Nav, CommandPalette,
 * PrincipiosList, LoveButton, ArticleBody highlights, theme, view transitions).
 */
(function () {
    'use strict';

    var doc = document;
    var win = window;

    function $(sel, root) {
        return (root || doc).querySelector(sel);
    }
    function $$(sel, root) {
        return Array.prototype.slice.call((root || doc).querySelectorAll(sel));
    }
    function pad(n) {
        return String(n).padStart(2, '0');
    }
    function isTypingTarget(el) {
        if (!el || typeof el.tagName !== 'string') return false;
        var tag = el.tagName.toLowerCase();
        return tag === 'input' || tag === 'textarea' || tag === 'select' || (el.isContentEditable === true);
    }
    function storageGet(key) {
        try {
            return win.localStorage.getItem(key);
        } catch (e) {
            return null;
        }
    }
    function storageSet(key, value) {
        try {
            win.localStorage.setItem(key, value);
        } catch (e) {
            /* storage full or unavailable — fail silently */
        }
    }

    /* ------------------------------------------------ lucide icons */
    if (win.lucide && win.lucide.createIcons) {
        win.lucide.createIcons();
    }

    /* ------------------------------------------------ theme */
    function toggleTheme() {
        var body = doc.body;
        var dark = body.classList.contains('theme-dark');
        body.classList.remove('theme-dark', 'theme-light');
        body.classList.add(dark ? 'theme-light' : 'theme-dark');
        storageSet('theme', dark ? 'light' : 'dark');
    }

    /* ------------------------------------------------ nav menu */
    var navEl = $('#primary-nav');
    var navToggle = $('[data-nav-toggle]');
    var navOpened = false;

    function setNavAnim(cls) {
        if (!navEl) return;
        navEl.classList.remove('nav-open-anim', 'nav-close-anim');
        void navEl.offsetWidth;
        if (cls) navEl.classList.add(cls);
    }
    function applyNav() {
        if (!navEl) return;
        navEl.setAttribute('data-opened', String(navOpened));
        if (navToggle) {
            navToggle.classList.toggle('cross', navOpened);
            navToggle.setAttribute('aria-expanded', String(navOpened));
        }
        setNavAnim(navOpened ? 'nav-open-anim' : 'nav-close-anim');
    }
    function toggleNav() {
        navOpened = !navOpened;
        applyNav();
    }
    if (navEl) {
        navEl.addEventListener('animationend', function (ev) {
            var cls = ev && ev.animationName;
            if (cls === 'rubberBand' || cls === 'flipOutY') {
                navEl.classList.remove('nav-open-anim', 'nav-close-anim');
            }
        });
    }
    if (navToggle) navToggle.addEventListener('click', toggleNav);

    /* ------------------------------------------------ command palette */
    var paletteRoot = $('#command-palette-root');
    var paletteOpen = false;
    var paletteInput = null;
    var paletteValue = '';
    var paletteArticles = null;
    var paletteResults = [];
    var paletteSelected = 0;
    var paletteResultsEl = null;

    var paletteCommands = [
        {
            hint: ':help \u2014 show this command list',
            match: function (v) { return v === ':help'; },
            run: function () {
                var items = paletteCommands.map(function (c) {
                    return '<li><code>' + c.hint + '</code></li>';
                }).join('');
                items += '<li><code>Type 2+ characters to search article titles (\u2191\u2193 navigate, Enter opens)</code></li>';
                openModal({
                    title: 'Commands',
                    body: '<div class="font-mono text-sm md:text-xs"><ul class="space-y-1">' + items + '</ul></div>'
                });
            }
        },
        {
            hint: ':q \u2014 close the palette',
            match: function (v) { return v === ':q'; },
            instant: true,
            run: closePalette
        },
        {
            hint: 'home \u2014 go to the home page',
            match: function (v) { return v.toLowerCase() === 'home'; },
            run: function () {
                doc.location.href = '/';
                closePalette();
            }
        },
        {
            hint: '#5 \u2014 jump to principle 5',
            match: function (v) { return /^#\d+$/.test(v); },
            run: function () {
                var n = Number(/^#(\d+)$/.exec((paletteValue || '').trim())[1]);
                if (doc.location.pathname === '/') {
                    openPrinciple(n);
                    setPrincipleHash(n);
                } else {
                    doc.location.href = '/#pr-' + pad(n);
                }
                closePalette();
            }
        }
    ];
    function matchPaletteCommand(value) {
        for (var i = 0; i < paletteCommands.length; i++) {
            if (paletteCommands[i].match(value)) return paletteCommands[i];
        }
        return null;
    }

    function loadPaletteArticles() {
        if (paletteArticles !== null) return;
        paletteArticles = [];
        fetch('/data/articles.json')
            .then(function (res) {
                if (!res.ok) throw new Error('bad status ' + res.status);
                return res.json();
            })
            .then(function (data) {
                paletteArticles = Array.isArray(data) ? data : [];
                updatePalette();
            })
            .catch(function () {
                paletteArticles = [];
            });
    }
    function searchArticles(value) {
        var q = value.toLowerCase();
        return paletteArticles.filter(function (a) {
            return a.title && a.title.toLowerCase().indexOf(q) !== -1;
        });
    }
    function renderPaletteResults() {
        if (!paletteResultsEl) return;
        paletteResultsEl.textContent = '';
        if (!paletteResults.length) return;
        var list = doc.createElement('ul');
        list.className = 'command-results';
        paletteResults.forEach(function (a, i) {
            var li = doc.createElement('li');
            li.className = 'command-result' + (i === paletteSelected ? ' selected' : '');
            li.textContent = a.title;
            li.addEventListener('click', function () {
                paletteSelected = i;
                runCommand();
            });
            list.appendChild(li);
        });
        paletteResultsEl.appendChild(list);
    }
    function updatePalette() {
        if (!paletteInput) return;
        var value = (paletteValue || '').trim();
        var hint = $('.command-hint', paletteRoot);
        var cmd = matchPaletteCommand(value);
        if (cmd) {
            if (cmd.instant) {
                cmd.run();
                return;
            }
            paletteResults = [];
            paletteSelected = 0;
            if (hint) hint.textContent = cmd.hint + ' \u00b7 enter';
        } else if (value.length >= 2) {
            paletteResults = searchArticles(value);
            paletteSelected = Math.max(0, Math.min(paletteSelected, paletteResults.length - 1));
            if (hint) hint.textContent = paletteResults.length + ' article' + (paletteResults.length === 1 ? '' : 's') + ' \u00b7 \u2191\u2193 navigate \u00b7 enter to open';
        } else {
            paletteResults = [];
            paletteSelected = 0;
            if (hint) hint.textContent = 'Jump to a principle (#5) or search articles';
        }
        renderPaletteResults();
    }

    function renderPalette() {
        if (!paletteRoot) return;
        paletteRoot.textContent = '';
        paletteInput = null;
        paletteResults = [];
        paletteSelected = 0;
        paletteResultsEl = null;
        if (!paletteOpen) return;

        var bar = doc.createElement('div');
        bar.className = 'command-palette';
        bar.setAttribute('role', 'dialog');
        bar.setAttribute('aria-label', 'Command line');

        var prompt = doc.createElement('span');
        prompt.className = 'command-prompt';
        prompt.textContent = '/';

        var input = doc.createElement('input');
        input.type = 'text';
        input.className = 'command-input';
        input.setAttribute('aria-label', 'Command');
        input.autocomplete = 'off';
        input.setAttribute('autocapitalize', 'off');
        input.setAttribute('spellcheck', 'false');
        input.addEventListener('input', function () {
            paletteValue = input.value;
            updatePalette();
        });

        var hint = doc.createElement('span');
        hint.className = 'command-hint';
        hint.textContent = 'Jump to a principle (#5) or search articles';

        var results = doc.createElement('div');
        results.className = 'command-results-wrap';
        paletteResultsEl = results;

        bar.appendChild(prompt);
        bar.appendChild(input);
        bar.appendChild(hint);
        bar.appendChild(results);
        paletteRoot.appendChild(bar);
        paletteInput = input;
        paletteValue = '';
        input.focus();
        loadPaletteArticles();
    }
    function openPalette() {
        paletteOpen = true;
        renderPalette();
    }
    function closePalette() {
        paletteOpen = false;
        renderPalette();
    }
    function runCommand() {
        var value = (paletteValue || '').trim();
        var cmd = matchPaletteCommand(value);
        if (cmd) {
            cmd.run();
            return;
        }
        var art = paletteResults[paletteSelected];
        if (art && art.slug) {
            doc.location.href = '/blog/' + art.slug;
            closePalette();
        }
    }
    function paletteSelect(delta) {
        if (!paletteResults.length) return;
        paletteSelected = (paletteSelected + delta + paletteResults.length) % paletteResults.length;
        renderPaletteResults();
    }

    /* ------------------------------------------------ modal */
    var modalRoot = $('#modal-root');
    var modalOpen = false;
    var modalTrigger = null;

    function openModal(opts) {
        if (!modalRoot) return;
        var title = opts.title || '';
        var body = opts.body || '';

        modalRoot.textContent = '';
        modalTrigger = doc.activeElement;

        var backdrop = doc.createElement('div');
        backdrop.className = 'modal-backdrop';
        backdrop.setAttribute('role', 'presentation');

        var dialog = doc.createElement('div');
        dialog.className = 'modal modal-in';
        dialog.setAttribute('role', 'dialog');
        dialog.setAttribute('aria-modal', 'true');
        if (title) dialog.setAttribute('aria-label', title);
        dialog.tabIndex = -1;

        var close = doc.createElement('button');
        close.type = 'button';
        close.className = 'modal-close';
        close.setAttribute('aria-label', 'Close');
        close.textContent = '\u2715';
        close.addEventListener('click', closeModal);

        if (title) {
            var h = doc.createElement('h2');
            h.className = 'modal-title';
            h.textContent = title;
            dialog.appendChild(h);
        }

        if (typeof body === 'string') {
            var p = doc.createElement('div');
            p.className = 'modal-body';
            p.innerHTML = body;
            dialog.appendChild(p);
        } else {
            body.className = (body.className || '') + ' modal-body';
            dialog.appendChild(body);
        }

        dialog.appendChild(close);
        backdrop.appendChild(dialog);
        modalRoot.appendChild(backdrop);

        backdrop.addEventListener('click', function (e) {
            if (e.target === backdrop) closeModal();
        });

        modalOpen = true;
        dialog.focus();
    }
    function closeModal() {
        if (!modalRoot || !modalOpen) return;
        modalRoot.textContent = '';
        modalOpen = false;
        if (modalTrigger && modalTrigger.focus) modalTrigger.focus();
        modalTrigger = null;
    }

    /* ------------------------------------------------ battery modal */
    $$('.battery-info').forEach(function (btn) {
        btn.addEventListener('click', function () {
            var avail = btn.getAttribute('data-available') === 'true';
            var rows = [];
            if (avail) {
                var pct = btn.getAttribute('data-percentage');
                var charging = btn.getAttribute('data-charging') === 'true';
                var status = btn.getAttribute('data-status') || '';
                var health = btn.getAttribute('data-health') || '';
                var plugged = btn.getAttribute('data-plugged') || '';
                var temp = Number(btn.getAttribute('data-temperature'));
                var current = Number(btn.getAttribute('data-current'));
                rows.push('<strong>' + pct + '%</strong> · ' + (charging ? 'Charging' : 'Not charging'));
                if (status) rows.push('Status: ' + status);
                if (health) rows.push('Health: ' + health);
                if (plugged) rows.push('Plugged: ' + plugged);
                if (!isNaN(temp)) rows.push('Temperature: ' + temp + '\u00b0C');
                if (!isNaN(current)) rows.push('Current: ' + current + ' mA');
            } else {
                rows.push('Battery status unavailable.');
            }
            var device = btn.getAttribute('data-device');
            var intro = device
                ? 'This site is being served from a <strong>' + device + ' Android Phone</strong> running the Go/Chi stack.'
                : 'This site is being served from an Android phone running the Go/Chi stack.';
            var body = '<p>' + intro + '</p><p class="opacity-70">' + rows.join('<br>') + '</p><p>Be self-soveraign ✊</p>';
            openModal({ title: 'Battery', body: body });
        });
    });

    /* ------------------------------------------------ principles */
    var principleScroll = $('#principle-scroll');
    var principleRows = $$('[data-principle-row]', principleScroll);
    var selected = 0;
    var open = null;

    function rowExists(n) {
        return principleRows.some(function (r) { return Number(r.getAttribute('data-principle-row')) === n; });
    }
    function hashFor(n) {
        return '#pr-' + pad(n);
    }
    function parseHash(hash) {
        var m = /^#pr-(\d+)$/.exec(hash);
        if (!m) return null;
        var n = Number(m[1]);
        return rowExists(n) ? n : null;
    }
    function setPrincipleHash(n) {
        var url = new URL(win.location.href);
        url.hash = n === null ? '' : hashFor(n);
        history.pushState(null, '', url.toString());
    }
    function detailFor(n) {
        return $('[data-principle-detail="' + n + '"]', principleScroll);
    }
    function applyPrincipleUI() {
        principleRows.forEach(function (row, i) {
            var n = Number(row.getAttribute('data-principle-row'));
            row.classList.toggle('selected', i === selected);
            row.classList.toggle('open', open === n);
            row.setAttribute('aria-expanded', String(open === n));
            var detail = detailFor(n);
            if (!detail) return;
            if (open === n) {
                if (detail.hidden) {
                    detail.hidden = false;
                    detail.classList.remove('pr-detail-anim');
                    void detail.offsetWidth;
                    detail.classList.add('pr-detail-anim');
                }
            } else {
                detail.hidden = true;
            }
        });
    }
    function openPrinciple(n) {
        var idx = principleRows.findIndex(function (r) { return Number(r.getAttribute('data-principle-row')) === n; });
        if (idx === -1) return;
        selected = idx;
        open = n;
        setPrincipleHash(n);
        applyPrincipleUI();
        focusPrincipleRow(idx);
        centerPrincipleDetail();
    }
    function togglePrinciple(n) {
        open = open === n ? null : n;
        setPrincipleHash(open);
        applyPrincipleUI();
        centerPrincipleDetail();
    }
    function focusPrincipleRow(i) {
        var row = principleRows[i];
        if (!row) return;
        row.focus({ preventScroll: true });
        row.scrollIntoView({ block: 'nearest' });
    }
    function centerPrincipleDetail() {
        if (open === null) return;
        var detail = detailFor(open);
        if (detail) {
            requestAnimationFrame(function () {
                requestAnimationFrame(function () {
                    detail.scrollIntoView({ behavior: 'smooth', block: 'center' });
                });
            });
        }
    }
    function focusInPrinciples(el) {
        if (!principleScroll || !el) return false;
        return principleScroll === el || principleScroll.contains(el);
    }
    function principleKeydown(e) {
        if (!focusInPrinciples(doc.activeElement) || !principleRows.length) return;
        if (e.key === 'ArrowDown') {
            e.preventDefault();
            selected = Math.min(selected + 1, principleRows.length - 1);
            focusPrincipleRow(selected);
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            selected = Math.max(selected - 1, 0);
            focusPrincipleRow(selected);
        } else if (e.key === 'Home') {
            e.preventDefault();
            selected = 0;
            focusPrincipleRow(selected);
        } else if (e.key === 'End') {
            e.preventDefault();
            selected = principleRows.length - 1;
            focusPrincipleRow(selected);
        } else if (e.key === 'Escape' && open !== null) {
            e.preventDefault();
            open = null;
            setPrincipleHash(null);
            applyPrincipleUI();
        }
    }
    if (principleRows.length) {
        principleRows.forEach(function (row) {
            row.addEventListener('click', function () {
                selected = principleRows.indexOf(row);
                togglePrinciple(Number(row.getAttribute('data-principle-row')));
            });
            row.addEventListener('focus', function () {
                selected = principleRows.indexOf(row);
                applyPrincipleUI();
            });
        });
        var applyHash = function () {
            var n = parseHash(win.location.hash);
            if (n !== null) {
                var idx = principleRows.findIndex(function (r) { return Number(r.getAttribute('data-principle-row')) === n; });
                if (idx >= 0) selected = idx;
                open = n;
                applyPrincipleUI();
                if (principleScroll) principleScroll.scrollIntoView({ block: 'nearest' });
                focusPrincipleRow(idx >= 0 ? idx : 0);
                centerPrincipleDetail();
            } else {
                open = null;
                applyPrincipleUI();
            }
        };
        applyHash();
        win.addEventListener('hashchange', applyHash);
    }

    /* ------------------------------------------------ love buttons */
    var VOTES_KEY = 'doags:votes:v1';
    function loadVotes() {
        var raw = storageGet(VOTES_KEY);
        if (!raw) return {};
        try {
            return JSON.parse(raw);
        } catch (e) {
            return {};
        }
    }
    function saveVotes(v) {
        storageSet(VOTES_KEY, JSON.stringify(v));
    }
    var PARTICLE_COLORS = ['#dc3545', '#ff6b81', '#ff9aa8', '#c81e3a'];

    function burst(x, y) {
        var count = 10;
        for (var i = 0; i < count; i++) {
            var el = doc.createElement('span');
            el.className = 'heart-particle';
            var angle = (Math.PI * 2 * i) / count + Math.random() * 0.4;
            var distance = 32 + Math.random() * 40;
            var tx = Math.cos(angle) * distance;
            var ty = Math.sin(angle) * distance - 10;
            el.style.setProperty('--tx', tx + 'px');
            el.style.setProperty('--ty', ty + 'px');
            el.style.left = x + 'px';
            el.style.top = y + 'px';
            el.style.background = PARTICLE_COLORS[i % PARTICLE_COLORS.length];
            doc.body.appendChild(el);
            (function (node) {
                setTimeout(function () {
                    if (node && node.parentNode) node.parentNode.removeChild(node);
                }, 700);
            })(el);
        }
    }
    function paintLove(btn, state, popping) {
        btn.classList.toggle('liked', state.liked);
        btn.setAttribute('aria-pressed', String(state.liked));
        btn.classList.toggle('border-hairline-strong', !state.liked);
        btn.classList.toggle('hover:border-[#dc3545]/50', !state.liked);
        btn.classList.toggle('border-[#dc3545]/40', state.liked);
        btn.classList.toggle('bg-[#dc3545]/5', state.liked);
        if (popping) {
            btn.classList.remove('animate-pop');
            void btn.offsetWidth;
            btn.classList.add('animate-pop');
            setTimeout(function () { btn.classList.remove('animate-pop'); }, 450);
        }
        var svg = btn.querySelector('svg');
        if (svg) {
            svg.classList.toggle('text-ink-soft', !state.liked);
            svg.setAttribute('fill', state.liked ? '#dc3545' : 'none');
            svg.setAttribute('stroke', state.liked ? '#dc3545' : 'currentColor');
            svg.setAttribute('color', state.liked ? '#dc3545' : 'currentColor');
        }
        var countEl = btn.querySelector('[data-count]');
        if (countEl) {
            countEl.textContent = String(state.count);
            countEl.classList.toggle('text-ink-soft', !state.liked);
            countEl.classList.toggle('text-[#dc3545]', state.liked);
        }
    }
    function wireLove(btn) {
        var slug = btn.getAttribute('data-slug');
        var base = Number(btn.getAttribute('data-base') || 0);
        var votes = loadVotes();
        var cur = votes[slug] || { liked: false, count: base };
        paintLove(btn, cur, false);
        btn.addEventListener('click', function () {
            var votesNow = loadVotes();
            var current = votesNow[slug] || { liked: false, count: base };
            var next = { liked: !current.liked, count: current.liked ? current.count - 1 : current.count + 1 };
            votesNow[slug] = next;
            saveVotes(votesNow);
            paintLove(btn, next, true);
            if (!current.liked) {
                var rect = btn.getBoundingClientRect();
                burst(rect.left + rect.width / 2, rect.top + rect.height / 2);
            }
        });
    }
    $$('[data-love]').forEach(wireLove);

    /* ------------------------------------------------ highlights */
    var HIGHLIGHTS_KEY = 'doags:highlights:v1';
    var toolbarEl = null;
    var popoverEl = null;
    var pendingRange = null;
    var currentPopover = null;

    function loadHighlights() {
        var raw = storageGet(HIGHLIGHTS_KEY);
        if (!raw) return {};
        try {
            return JSON.parse(raw);
        } catch (e) {
            return {};
        }
    }
    function saveHighlights(h) {
        storageSet(HIGHLIGHTS_KEY, JSON.stringify(h));
    }
    function forSlug(slug) {
        var all = loadHighlights();
        return all[slug] || [];
    }
    function makeId() {
        if (win.crypto && win.crypto.randomUUID) return win.crypto.randomUUID();
        return Date.now() + '-' + Math.random().toString(36).slice(2, 9);
    }
    function wrapRange(range, id) {
        var mark = doc.createElement('mark');
        mark.className = 'highlight';
        mark.setAttribute('data-highlight-id', id);
        try {
            range.surroundContents(mark);
            return true;
        } catch (err) {
            try {
                var contents = range.extractContents();
                mark.appendChild(contents);
                range.insertNode(mark);
                return true;
            } catch (err2) {
                return false;
            }
        }
    }
    function hideToolbar() {
        if (toolbarEl && toolbarEl.parentNode) toolbarEl.parentNode.removeChild(toolbarEl);
        toolbarEl = null;
        pendingRange = null;
    }
    function hidePopover() {
        if (popoverEl && popoverEl.parentNode) popoverEl.parentNode.removeChild(popoverEl);
        popoverEl = null;
        currentPopover = null;
    }
    function renderHighlightsSection(slug) {
        var section = $('[data-your-highlights]');
        if (!section) return;
        var list = forSlug(slug);
        if (!list.length) {
            section.hidden = true;
            return;
        }
        section.hidden = false;
        var count = $('[data-highlights-count]', section);
        if (count) count.textContent = String(list.length);
        var ul = $('[data-highlights-list]', section);
        if (!ul) return;
        ul.textContent = '';
        list.forEach(function (h) {
            var li = doc.createElement('li');
            li.className = 'border-l-2 pl-3';
            li.style.borderColor = 'var(--color-mark)';
            var quote = doc.createElement('p');
            quote.className = 'font-body text-[14px] text-ink-soft italic';
            quote.textContent = '"' + h.quote + '"';
            li.appendChild(quote);
            if (h.note) {
                var note = doc.createElement('p');
                note.className = 'mt-1 font-mono text-[12px] text-muted';
                note.textContent = h.note;
                li.appendChild(note);
            }
            ul.appendChild(li);
        });
    }
    function findAndWrap(container, quote, id) {
        var walker = doc.createTreeWalker(container, NodeFilter.SHOW_TEXT);
        var nodes = [];
        var full = '';
        var n = null;
        while ((n = walker.nextNode())) {
            var t = n;
            var start = full.length;
            full += t.data;
            nodes.push({ node: t, start: start, end: full.length });
        }
        var idx = full.indexOf(quote);
        if (idx === -1) return;
        var endIdx = idx + quote.length;
        var startInfo = null;
        var endInfo = null;
        for (var i = 0; i < nodes.length; i++) {
            if (idx >= nodes[i].start && idx < nodes[i].end) startInfo = nodes[i];
            if (endIdx > nodes[i].start && endIdx <= nodes[i].end) endInfo = nodes[i];
        }
        if (!startInfo || !endInfo) return;
        var range = doc.createRange();
        range.setStart(startInfo.node, idx - startInfo.start);
        range.setEnd(endInfo.node, endIdx - endInfo.start);
        wrapRange(range, id);
    }

    var bodyEl = $('[data-article-body]');
    if (bodyEl) {
        var slug = bodyEl.getAttribute('data-article-slug');
        renderHighlightsSection(slug);
        forSlug(slug).forEach(function (h) {
            if (bodyEl.querySelector('mark[data-highlight-id="' + h.id + '"]')) return;
            findAndWrap(bodyEl, h.quote, h.id);
        });

        doc.addEventListener('mouseup', function () {
            var sel = win.getSelection();
            if (!sel || sel.isCollapsed || !bodyEl) {
                hideToolbar();
                return;
            }
            var range = sel.getRangeAt(0);
            if (!bodyEl.contains(range.commonAncestorContainer)) {
                hideToolbar();
                return;
            }
            var text = range.toString().trim();
            if (!text) {
                hideToolbar();
                return;
            }
            var rect = range.getBoundingClientRect();
            pendingRange = range.cloneRange();
            hidePopover();
            hideToolbar();
            toolbarEl = doc.createElement('button');
            toolbarEl.type = 'button';
            toolbarEl.className = 'fixed z-30 -translate-x-1/2 -translate-y-[calc(100%+10px)] rounded-sm bg-ink px-3 py-1.5 font-mono text-[11px] tracking-wide text-paper shadow-lg transition-transform';
            toolbarEl.style.left = Math.round(rect.left + rect.width / 2) + 'px';
            toolbarEl.style.top = Math.round(rect.top) + 'px';
            toolbarEl.textContent = 'Highlight';
            toolbarEl.addEventListener('click', function () {
                var quote2 = pendingRange ? pendingRange.toString().trim() : '';
                if (!quote2) return;
                var h = addHighlight(slug, quote2);
                if (pendingRange) {
                    var ok = wrapRange(pendingRange, h.id);
                    if (!ok) removeStored(slug, h.id);
                }
                win.getSelection().removeAllRanges();
                hideToolbar();
                renderHighlightsSection(slug);
            });
            doc.body.appendChild(toolbarEl);
        });

        bodyEl.addEventListener('click', function (e) {
            var target = e.target;
            var mark = target.closest ? target.closest('mark.highlight') : null;
            if (!mark || !bodyEl) return;
            var id = mark.getAttribute('data-highlight-id') || '';
            var all = loadHighlights();
            var list = all[slug] || [];
            var found = null;
            for (var i = 0; i < list.length; i++) {
                if (list[i].id === id) found = list[i];
            }
            if (!found) return;
            var rect = mark.getBoundingClientRect();
            hideToolbar();
            hidePopover();
            currentPopover = { id: id, note: found.note };
            popoverEl = doc.createElement('div');
            popoverEl.className = 'fixed z-30 w-64 -translate-x-1/2 rounded-sm border border-hairline-strong bg-paper p-3 shadow-xl';
            popoverEl.style.left = Math.round(rect.left + rect.width / 2) + 'px';
            popoverEl.style.top = Math.round(rect.bottom + 8) + 'px';

            var label = doc.createElement('p');
            label.className = 'font-mono text-[10px] tracking-wide text-faint uppercase';
            label.textContent = 'Your note';

            var textarea = doc.createElement('textarea');
            textarea.rows = 3;
            textarea.placeholder = 'Why did this line stop you?';
            textarea.className = 'mt-2 w-full resize-none border border-hairline bg-paper-dim px-2 py-1.5 font-body text-[13px] text-ink-soft outline-none focus:border-ink';
            textarea.value = found.note || '';
            textarea.addEventListener('input', function () {
                if (currentPopover) currentPopover.note = textarea.value;
            });

            var actions = doc.createElement('div');
            actions.className = 'mt-2 flex items-center justify-between';

            var removeBtn = doc.createElement('button');
            removeBtn.type = 'button';
            removeBtn.className = 'font-mono text-[11px] text-muted hover:text-ink';
            removeBtn.textContent = 'Remove';
            removeBtn.addEventListener('click', function () {
                if (currentPopover) removeStored(slug, currentPopover.id);
                hidePopover();
                renderHighlightsSection(slug);
            });

            var right = doc.createElement('div');
            right.className = 'flex gap-2';

            var closeBtn = doc.createElement('button');
            closeBtn.type = 'button';
            closeBtn.className = 'font-mono text-[11px] text-muted hover:text-ink';
            closeBtn.textContent = 'Close';
            closeBtn.addEventListener('click', hidePopover);

            var saveBtn = doc.createElement('button');
            saveBtn.type = 'button';
            saveBtn.className = 'bg-ink px-2 py-1 font-mono text-[11px] text-paper';
            saveBtn.textContent = 'Save';
            saveBtn.addEventListener('click', function () {
                if (currentPopover) {
                    var all2 = loadHighlights();
                    var list2 = all2[slug] || [];
                    all2[slug] = list2.map(function (h) {
                        return h.id === currentPopover.id ? { id: h.id, quote: h.quote, note: currentPopover.note || '', createdAt: h.createdAt } : h;
                    });
                    saveHighlights(all2);
                    renderHighlightsSection(slug);
                }
                hidePopover();
            });

            right.appendChild(closeBtn);
            right.appendChild(saveBtn);
            actions.appendChild(removeBtn);
            actions.appendChild(right);
            popoverEl.appendChild(label);
            popoverEl.appendChild(textarea);
            popoverEl.appendChild(actions);
            doc.body.appendChild(popoverEl);
        });

        doc.addEventListener('scroll', function () {
            hideToolbar();
            hidePopover();
        }, true);
    }

    function addHighlight(slug, quote) {
        var h = { id: makeId(), quote: quote, note: '', createdAt: new Date().toISOString() };
        var all = loadHighlights();
        all[slug] = (all[slug] || []).concat([h]);
        saveHighlights(all);
        return h;
    }
    function removeStored(slug, id) {
        var all = loadHighlights();
        var list = all[slug] || [];
        all[slug] = list.filter(function (h) { return h.id !== id; });
        saveHighlights(all);
    }

    /* ------------------------------------------------ keyboard */
    doc.addEventListener('keydown', function (e) {
        var target = e.target;
        if (e.key === 'Escape' && navOpened) toggleNav();

        /* modal */
        if (e.key === 'Escape' && modalOpen) {
            e.preventDefault();
            closeModal();
            return;
        }

        /* command palette */
        if (e.metaKey || e.ctrlKey || e.altKey) return;
        if (e.key === '/' && !paletteOpen && !isTypingTarget(target)) {
            e.preventDefault();
            openPalette();
            return;
        }
        if (paletteOpen && !modalOpen) {
            if (e.key === 'Enter') {
                e.preventDefault();
                runCommand();
            } else if (e.key === 'Escape') {
                e.preventDefault();
                closePalette();
            } else if (e.key === 'ArrowDown') {
                e.preventDefault();
                paletteSelect(1);
            } else if (e.key === 'ArrowUp') {
                e.preventDefault();
                paletteSelect(-1);
            }
            return;
        }
        if (isTypingTarget(target)) return;
        if (e.key === 'm') toggleNav();
        if (e.key === 'd') toggleTheme();
        principleKeydown(e);
    });

    /* ------------------------------------------------ theme toggle buttons */
    $$('.theme-toggle').forEach(function (btn) {
        btn.addEventListener('click', toggleTheme);
    });

    /* ------------------------------------------------ login form */
    var loginForm = $('[data-login-form]');
    if (loginForm) {
        loginForm.addEventListener('submit', function (e) {
            e.preventDefault();
            var email = '';
            var emailInput = loginForm.querySelector('#email');
            if (emailInput) email = emailInput.value;
            var p = doc.createElement('p');
            p.className = 'mt-8 border border-hairline bg-paper-dim px-4 py-3 font-mono text-[12px] text-ink-soft';
            p.textContent = "We'll email a link to " + email + " the day this actually works.";
            loginForm.parentNode.replaceChild(p, loginForm);
        });
    }

    /* ------------------------------------------------ injected micro-animations */
    var style = doc.createElement('style');
    style.textContent =
        '.pr-detail-anim{animation:prSlide .3s var(--ease-shizuka, cubic-bezier(.22,1,.36,1)) both;}' +
        '@keyframes prSlide{from{opacity:0;transform:translateY(-8px)}to{opacity:1;transform:none}}';
    doc.head.appendChild(style);

    /* ------------------------------------------------ view transitions */
    doc.addEventListener('click', function (e) {
        var a = null;
        if (e.target && e.target.closest) a = e.target.closest('a');
        if (!a) return;
        if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey || e.defaultPrevented) return;
        var url;
        try {
            url = new URL(a.href);
        } catch (err) {
            return;
        }
        if (url.origin !== win.location.origin) return;
        if (url.pathname === win.location.pathname && url.hash) return;
        e.preventDefault();
        if (doc.startViewTransition) {
            doc.startViewTransition(function () {
                win.location.href = url.href;
            });
        } else {
            win.location.href = url.href;
        }
    });
})();
