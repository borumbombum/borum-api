// img2webp experiment script: shows the picked file name next to the upload,
// syncs the quality slider label, and drives the convert form with a progress
// bar. The form posts via XHR so the button can disable during conversion and
// the upload percentage is shown; the returned WebP downloads like a normal
// file. Errors surface in a status line and re-enable the form.
(function () {
    var form = document.querySelector('[data-convert-form]');
    if (!form) return;

    var input = form.querySelector('input[name="image"]');
    var name = document.querySelector('[data-file-name]');
    var slider = form.querySelector('input[name="quality"]');
    var value = document.querySelector('[data-quality-value]');
    var submit = document.querySelector('[data-convert-submit]');
    var label = document.querySelector('[data-convert-label]');
    var spinner = document.querySelector('[data-spinner]');
    var progress = document.querySelector('[data-progress]');
    var fill = document.querySelector('[data-progress-fill]');
    var progressLabel = document.querySelector('[data-progress-label]');
    var status = document.querySelector('[data-convert-status]');

    if (input && name) {
        input.addEventListener('change', function () {
            var f = input.files && input.files[0];
            name.textContent = f ? f.name + ' \u2014 ' + (f.size / 1024 / 1024).toFixed(2) + ' MB' : '';
        });
    }

    if (slider && value) {
        slider.addEventListener('input', function () {
            value.textContent = slider.value;
        });
    }

    form.addEventListener('submit', function (e) {
        e.preventDefault();
        hideStatus();
        var file = input.files && input.files[0];
        if (!file) {
            showStatus('choose a PNG or JPEG image first');
            return;
        }

        setBusy(true);
        var xhr = new XMLHttpRequest();
        xhr.open('POST', form.action);
        xhr.responseType = 'blob';

        xhr.upload.addEventListener('progress', function (ev) {
            if (ev.lengthComputable) {
                var pct = Math.round((ev.loaded / ev.total) * 100);
                showProgress(pct, 'uploading\u2026 ' + pct + '%');
            }
        });

        xhr.addEventListener('load', function () {
            if (xhr.status >= 200 && xhr.status < 300) {
                var url = URL.createObjectURL(xhr.response);
                var a = document.createElement('a');
                a.href = url;
                a.download = downloadName(xhr);
                document.body.appendChild(a);
                a.click();
                a.remove();
                URL.revokeObjectURL(url);
                reset();
            } else {
                setBusy(false);
                showStatus(xhr.responseText || 'could not convert this image');
            }
        });

        xhr.addEventListener('error', function () {
            setBusy(false);
            showStatus('network error \u2014 could not reach the server');
        });

        xhr.send(new FormData(form));
    });

    function setBusy(b) {
        submit.disabled = b;
        label.textContent = b ? 'converting\u2026' : 'convert & download';
        if (spinner) spinner.classList.toggle('hidden', !b);
        if (fill) fill.classList.toggle('animate-pulse', b);
    }

    function showProgress(pct, text) {
        if (progress) progress.classList.remove('hidden');
        fill.style.width = pct + '%';
        progressLabel.textContent = text;
    }

    function showStatus(msg) {
        status.textContent = msg;
        status.classList.remove('hidden');
    }

    function hideStatus() {
        if (status) status.classList.add('hidden');
    }

    function reset() {
        if (progress) progress.classList.add('hidden');
        fill.style.width = '0%';
        progressLabel.textContent = '';
        hideStatus();
        setBusy(false);
    }

    // Prefer the server's attachment filename; fall back to the upload name.
    function downloadName(xhr) {
        var cd = xhr.getResponseHeader('Content-Disposition') || '';
        var m = cd.match(/filename="([^"]+)"/);
        if (m) return m[1];
        var f = input.files && input.files[0] ? input.files[0].name : 'image.webp';
        return f.replace(/\.[^.]+$/, '') + '.webp';
    }
})();
