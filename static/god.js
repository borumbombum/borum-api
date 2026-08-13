/* Admin (/god) behaviour: drawer toggle and logout. The article forms are
   plain HTML posts, so this file stays small. */
(function () {
    'use strict';
    var doc = document;

    /* ------------------------------------------------ drawer */
    var drawer = doc.querySelector('[data-drawer]');
    var backdrop = doc.querySelector('[data-drawer-backdrop]');

    function closeDrawer() {
        if (!drawer) return;
        drawer.classList.add('-translate-x-full');
        if (backdrop) backdrop.classList.add('hidden');
    }

    doc.querySelectorAll('[data-drawer-toggle]').forEach(function (btn) {
        btn.addEventListener('click', function () {
            if (drawer.classList.contains('-translate-x-full')) {
                drawer.classList.remove('-translate-x-full');
                if (backdrop) backdrop.classList.remove('hidden');
            } else {
                closeDrawer();
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
