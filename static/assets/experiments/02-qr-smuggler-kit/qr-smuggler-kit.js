/* QR Smuggler Kit — client-side QR decode/encode.
   All processing happens in the browser; nothing is uploaded to the server. */
(function () {
    'use strict';
    var doc = document;

    /* ------------------------------------------------ tab switching */
    var tabs = doc.querySelectorAll('[data-tab]');
    var panels = doc.querySelectorAll('[data-panel]');

    tabs.forEach(function (tab) {
        tab.addEventListener('click', function () {
            var target = tab.getAttribute('data-tab');
            tabs.forEach(function (t) {
                var isActive = t.getAttribute('data-tab') === target;
                t.classList.toggle('font-semibold', isActive);
                t.classList.toggle('text-ink', isActive);
                t.classList.toggle('border-ink', isActive);
                t.classList.toggle('text-faint', !isActive);
                t.classList.toggle('border-transparent', !isActive);
            });
            panels.forEach(function (p) {
                p.classList.toggle('hidden', p.getAttribute('data-panel') !== target);
            });
        });
    });

    /* ------------------------------------------------ helpers */
    var canvas = doc.createElement('canvas');
    var ctx = canvas.getContext('2d');

    function decodeFromImageData(imageData, width, height) {
        var code = jsQR(imageData.data, width, height);
        return code ? code.data : null;
    }

    function drawImageToCanvas(img) {
        canvas.width = img.naturalWidth || img.width;
        canvas.height = img.naturalHeight || img.height;
        ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
        return ctx.getImageData(0, 0, canvas.width, canvas.height);
    }

    function showDecodeResult(text) {
        var el = doc.getElementById('decode-result');
        if (!text) {
            el.innerHTML = '<p class="font-mono text-[12px] text-faint">No QR code found in this image.</p>';
            return;
        }
        el.innerHTML =
            '<p class="mb-2 font-mono text-[12px] text-faint">Decoded content:</p>' +
            '<pre class="mb-3 max-h-60 overflow-auto whitespace-pre-wrap break-all rounded bg-soft p-3 font-mono text-[12px]"></pre>' +
            '<button type="button" data-copy-qr class="cursor-pointer rounded border border-hairline px-3 py-1 font-mono text-[12px] text-faint hover:text-ink">Copy</button>';
        el.querySelector('pre').textContent = text;
        el.querySelector('[data-copy-qr]').addEventListener('click', function () {
            navigator.clipboard.writeText(text).then(function () {
                var btn = el.querySelector('[data-copy-qr]');
                btn.textContent = 'Copied';
                setTimeout(function () { btn.textContent = 'Copy'; }, 1500);
            });
        });
    }

    /* ------------------------------------------------ decode: file upload */
    var fileInput = doc.getElementById('qr-file-input');
    if (fileInput) {
        fileInput.addEventListener('change', function () {
            var file = fileInput.files && fileInput.files[0];
            if (!file) return;
            var reader = new FileReader();
            reader.onload = function (e) {
                var img = new Image();
                img.onload = function () {
                    var data = drawImageToCanvas(img);
                    showDecodeResult(decodeFromImageData(data, canvas.width, canvas.height));
                };
                img.src = e.target.result;
            };
            reader.readAsDataURL(file);
        });
    }

    /* ------------------------------------------------ decode: camera */
    var cameraBtn = doc.getElementById('qr-camera-btn');
    var captureBtn = doc.getElementById('qr-capture-btn');
    var cameraWrap = doc.getElementById('camera-wrap');
    var video = doc.getElementById('qr-camera');
    var cameraStream = null;

    if (cameraBtn) {
        cameraBtn.addEventListener('click', function () {
            if (cameraStream) {
                cameraStream.getTracks().forEach(function (t) { t.stop(); });
                cameraStream = null;
                cameraWrap.classList.add('hidden');
                cameraBtn.textContent = 'Open Camera';
                return;
            }
            if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
                alert('Camera access is not supported in this browser.');
                return;
            }
            navigator.mediaDevices.getUserMedia({
                video: { facingMode: 'environment' }
            }).then(function (stream) {
                cameraStream = stream;
                video.srcObject = stream;
                cameraWrap.classList.remove('hidden');
                cameraBtn.textContent = 'Close Camera';
            }).catch(function () {
                alert('Camera permission denied or unavailable.');
            });
        });
    }

    if (captureBtn) {
        captureBtn.addEventListener('click', function () {
            if (!video.videoWidth) return;
            canvas.width = video.videoWidth;
            canvas.height = video.videoHeight;
            ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
            var data = ctx.getImageData(0, 0, canvas.width, canvas.height);
            showDecodeResult(decodeFromImageData(data, canvas.width, canvas.height));
        });
    }

    /* ------------------------------------------------ encode (auto on input) */
    var textInput = doc.getElementById('qr-text');
    var encodeTimer = null;

    function doEncode() {
        var text = textInput.value.trim();
        var el = doc.getElementById('encode-result');
        if (!text) {
            el.innerHTML = '<p class="font-mono text-[12px] text-faint">QR image will appear here.</p>';
            return;
        }
        try {
            var qr = qrcode(0, 'M');
            qr.addData(text);
            qr.make();
            el.innerHTML =
                '<div class="flex flex-col items-center gap-3">' +
                qr.createImgTag(4, 4) +
                '<a download="qr.png" class="cursor-pointer rounded border border-hairline px-3 py-1 font-mono text-[12px] text-faint hover:text-ink">Download PNG</a>' +
                '</div>';
            var img = el.querySelector('img');
            if (img) {
                var dlLink = el.querySelector('a[download]');
                dlLink.href = img.src;
            }
        } catch (e) {
            el.innerHTML = '<p class="font-mono text-[12px] text-red-600">Failed to encode: ' + e.message + '</p>';
        }
    }

    if (textInput) {
        textInput.addEventListener('input', function () {
            clearTimeout(encodeTimer);
            encodeTimer = setTimeout(doEncode, 200);
        });
    }
})();
