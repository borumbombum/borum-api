// Entry point for the vendored TipTap bundle. Rebuilds run through this file
// so the exported global stays stable across version bumps.
import { Editor } from '@tiptap/core';
import StarterKit from '@tiptap/starter-kit';
import Link from '@tiptap/extension-link';
import Image from '@tiptap/extension-image';
import Placeholder from '@tiptap/extension-placeholder';

export { Editor, StarterKit, Link, Image, Placeholder };
