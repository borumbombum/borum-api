/* Admin (/god) behaviour: drawer, logout, and modal. The article forms are
   plain HTML posts, so this file stays small. */
(function () {
    'use strict';
    var doc = document;
    var win = window;

    /* ------------------------------------------------ lucide icons */
    function mountIcons() {
        if (win.lucide && win.lucide.createIcons) {
            win.lucide.createIcons();
        }
    }
    mountIcons();
    if (doc.readyState === 'loading') {
        doc.addEventListener('DOMContentLoaded', mountIcons);
    }

    /* ------------------------------------------------ drawer */
    var drawer = doc.querySelector('[data-drawer]');
    var backdrop = doc.querySelector('[data-drawer-backdrop]');

    function closeDrawer() {
        if (!drawer) return;
        drawer.style.transform = 'translateX(-100%)';
        if (backdrop) backdrop.hidden = true;
    }

    doc.querySelectorAll('[data-drawer-toggle]').forEach(function (btn) {
        btn.addEventListener('click', function () {
            if (drawer.style.transform === 'translateX(0%)') {
                closeDrawer();
            } else {
                drawer.style.transform = 'translateX(0%)';
                if (backdrop) backdrop.hidden = false;
            }
        });
    });
    if (backdrop) backdrop.addEventListener('click', closeDrawer);

    /* ------------------------------------------------ modal */
    var modalRoot = doc.getElementById('modal-root');
    var modalOpen = false;
    var modalTrigger = null;

    win.openModal = function (opts) {
        if (!modalRoot) return;
        var title = opts.title || '';
        var body = opts.body || '';

        modalRoot.textContent = '';
        modalTrigger = doc.activeElement;

        var bd = doc.createElement('div');
        bd.className = 'modal-backdrop';
        bd.setAttribute('role', 'presentation');

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
        close.addEventListener('click', win.closeModal);

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
        bd.appendChild(dialog);
        modalRoot.appendChild(bd);

        bd.addEventListener('click', function (e) {
            if (e.target === bd) win.closeModal();
        });

        modalOpen = true;
        dialog.focus();
    };

    win.closeModal = function () {
        if (!modalRoot || !modalOpen) return;
        modalRoot.textContent = '';
        modalOpen = false;
        if (modalTrigger && modalTrigger.focus) modalTrigger.focus();
        modalTrigger = null;
    };

    doc.addEventListener('keydown', function (e) {
        if (e.key === 'Escape' && modalOpen) {
            e.preventDefault();
            win.closeModal();
        }
    });

    /* ------------------------------------------------ logout */
    var logout = doc.querySelector('[data-logout]');
    if (logout) {
        logout.addEventListener('click', function () {
            fetch('/api/v1/auth/logout', { method: 'POST' }).then(function () {
                window.location.href = '/login';
            });
        });
    }
})();
