/* god-editor.js — TipTap rich-text editor for admin content fields (article
   body, experiment intro). Every [data-editor] mount on the page gets its own
   editor and toolbar; the mount's data-editor attribute names the hidden
   textarea it syncs to (unique per page). Registers no globals. If the TipTap
   bundle is missing it falls back to the plain textarea, so the form always
   works. */
(function () {
    'use strict';
    var doc = document;
    var win = window;

    function initEditor(mount) {
        var textarea = doc.getElementById(mount.getAttribute('data-editor'));
        if (!textarea || !textarea.form) return;
        var toolbar = mount.parentNode.querySelector('.editor-toolbar');

        // Bundle missing (deploy without the vendor files): keep the raw
        // textarea usable instead of a dead editor shell.
        if (!win.TipTap || !win.TipTap.Editor) {
            textarea.classList.remove('hidden');
            if (toolbar) toolbar.hidden = true;
            return;
        }

        var TipTap = win.TipTap;
        var sourceMode = false;

        /* ------------------------------------------------ toolbar */
        if (!toolbar) {
            toolbar = mount.parentNode.insertBefore(doc.createElement('div'), mount);
            toolbar.className = 'editor-toolbar';
        }

        var ICON = {
            bold: 'Bold', italic: 'Italic', underline: 'Underline', strike: 'Strikethrough',
            h1: 'Heading1', h2: 'Heading2', h3: 'Heading3',
            bulletList: 'List', orderedList: 'ListOrdered',
            blockquote: 'Quote', codeBlock: 'Code',
            link: 'Link', image: 'Image',
            undo: 'Undo2', redo: 'Redo2',
            source: 'FileCode'
        };
        var active = [
            ['bold'], ['italic'], ['underline'], ['strike'],
            ['heading', 1], ['heading', 2], ['heading', 3],
            ['bulletList'], ['orderedList'], ['blockquote'], ['codeBlock'],
            ['link']
        ];

        function icon(name) {
            var i = doc.createElement('i');
            i.setAttribute('data-lucide', name);
            i.setAttribute('width', '16');
            i.setAttribute('height', '16');
            return i;
        }

        function button(label, title, run) {
            var btn = doc.createElement('button');
            btn.type = 'button';
            btn.title = title;
            btn.setAttribute('aria-label', title);
            btn.setAttribute('data-editor-cmd', label);
            btn.appendChild(icon(ICON[label]));
            btn.addEventListener('click', function () { run(editor, btn); });
            toolbar.appendChild(btn);
            return btn;
        }

        function sep() {
            var s = doc.createElement('span');
            s.className = 'editor-toolbar-sep';
            s.setAttribute('aria-hidden', 'true');
            toolbar.appendChild(s);
        }

        function heading(level, label, title) {
            button(label, title, function (e) {
                e.chain().focus().toggleHeading({ level: level }).run();
            });
        }

        /* ------------------------------------------------ editor */
        var editor = new TipTap.Editor({
            element: mount,
            extensions: [
                TipTap.StarterKit.configure({
                    heading: { levels: [1, 2, 3] },
                    link: {
                        openOnClick: false,
                        HTMLAttributes: { rel: 'noopener noreferrer', target: '_blank' }
                    }
                }),
                TipTap.Image.configure({ inline: false }),
                TipTap.Placeholder.configure({ placeholder: 'write…' })
            ],
            content: textarea.value || '',
            onUpdate: function () {
                textarea.value = editor.getHTML();
            }
        });
        mount.classList.remove('hidden');

        /* ------------------------------------------------ buttons */
        button('bold', 'bold', function (e) { e.chain().focus().toggleBold().run(); });
        button('italic', 'italic', function (e) { e.chain().focus().toggleItalic().run(); });
        button('underline', 'underline', function (e) { e.chain().focus().toggleUnderline().run(); });
        button('strike', 'strike', function (e) { e.chain().focus().toggleStrike().run(); });
        sep();
        heading(1, 'h1', 'heading 1');
        heading(2, 'h2', 'heading 2');
        heading(3, 'h3', 'heading 3');
        sep();
        button('bulletList', 'bullet list', function (e) { e.chain().focus().toggleBulletList().run(); });
        button('orderedList', 'ordered list', function (e) { e.chain().focus().toggleOrderedList().run(); });
        button('blockquote', 'blockquote', function (e) { e.chain().focus().toggleBlockquote().run(); });
        button('codeBlock', 'code block', function (e) { e.chain().focus().toggleCodeBlock().run(); });
        sep();
        button('link', 'link', function (e) {
            var href = win.prompt('Link URL (https://…), empty to remove');
            if (href === null) return;
            if (href === '') {
                e.chain().focus().extendMarkRange('link').unsetLink().run();
            } else {
                e.chain().focus().extendMarkRange('link').setLink({ href: href }).run();
            }
        });
        button('image', 'image', function (e) {
            var src = win.prompt('Image URL');
            if (src) e.chain().focus().setImage({ src: src }).run();
        });
        sep();
        button('undo', 'undo', function (e) { e.chain().focus().undo().run(); });
        button('redo', 'redo', function (e) { e.chain().focus().redo().run(); });
        sep();
        button('source', 'edit html', toggleSource);

        // Source mode swaps the WYSIWYG view for the raw textarea. The form
        // textarea is always the source of truth, so submit needs no handler.
        function toggleSource() {
            sourceMode = !sourceMode;
            if (sourceMode) {
                textarea.value = editor.getHTML();
                mount.classList.add('hidden');
                textarea.classList.remove('hidden');
                textarea.focus();
            } else {
                editor.commands.setContent(textarea.value, { emitUpdate: true });
                textarea.classList.add('hidden');
                mount.classList.remove('hidden');
            }
        }

        /* ------------------------------------------------ active state */
        editor.on('transaction', function () {
            active.forEach(function (a) {
                var label = a[0] === 'heading' ? 'h' + a[1] : a[0];
                var isActive = a[0] === 'heading'
                    ? editor.isActive('heading', { level: a[1] })
                    : editor.isActive(a[0]);
                var btn = toolbar.querySelector('[data-editor-cmd="' + label + '"]');
                if (btn) btn.classList.toggle('is-active', isActive);
            });
            var src = toolbar.querySelector('[data-editor-cmd="source"]');
            if (src) src.classList.toggle('is-active', sourceMode);
        });

    }

    function init() {
        var mounts = doc.querySelectorAll('[data-editor]');
        Array.prototype.forEach.call(mounts, initEditor);
        if (win.lucide && win.lucide.createIcons) {
            win.lucide.createIcons();
        }
    }

    if (doc.readyState === 'loading') {
        doc.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
