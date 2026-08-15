/* BorumLoader — global top loading indicator.
 * Self-contained and dependency-free. Exposes window.BorumLoader.show() and
 * .hide(). The bar is created lazily on first show(), so pages that never
 * navigate pay nothing. Every call is a safe no-op when the module is missing.
 * Wired once: document-level click + submit listeners handle all same-origin
 * navigations; pageshow hides the bar so back/forward (bfcache) never leaves a
 * stale bar. */
(function () {
    'use strict';
    var doc = document;
    var win = window;

    var barEl = null;

    function show() {
        if (barEl) return;
        var bar = doc.createElement('div');
        bar.className = 'borum-loader';
        bar.setAttribute('aria-hidden', 'true');
        var pill = doc.createElement('span');
        pill.className = 'borum-loader-pill';
        bar.appendChild(pill);
        doc.body.appendChild(bar);
        barEl = bar;
    }
    function hide() {
        if (!barEl) return;
        if (barEl.parentNode) barEl.parentNode.removeChild(barEl);
        barEl = null;
    }

    function isNavigationClick(e) {
        if (e.defaultPrevented) return false;
        if (e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return false;
        var t = e.target;
        if (!t || !t.closest) return false;
        var a = t.closest('a[href]');
        if (!a) return false;
        if (a.hasAttribute('download')) return false;
        if ((a.getAttribute('target') || '').toLowerCase() === '_blank') return false;
        var u;
        try {
            u = new URL(a.href);
        } catch (err) {
            return false;
        }
        if (u.origin !== win.location.origin) return false;
        if (u.pathname + u.search === win.location.pathname + win.location.search) return false;
        return true;
    }

    doc.addEventListener('click', function (e) {
        if (isNavigationClick(e)) show();
    });

    doc.addEventListener('submit', function (e) {
        var form = e.target;
        if (!form || form.tagName !== 'FORM') return;
        if (e.defaultPrevented) return;
        if (form.matches && form.matches('[data-convert-form]')) return;
        var u;
        try {
            u = new URL(form.getAttribute('action') || win.location.href, win.location.href);
        } catch (err) {
            return;
        }
        if (u.origin === win.location.origin) show();
    });

    win.addEventListener('pageshow', hide);

    win.BorumLoader = { show: show, hide: hide };
})();
