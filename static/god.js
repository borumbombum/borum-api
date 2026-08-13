/* Admin (/god) behaviour: drawer toggle and logout. The article forms are
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
