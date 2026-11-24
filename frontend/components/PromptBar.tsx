'use client';

import React, {
  isValidElement,
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
  type PointerEvent,
  type ReactNode
} from 'react';
import { animate, useMotionValue, useMotionValueEvent, useReducedMotion } from 'motion/react';
import { HugeiconsIcon, type IconSvgElement } from '@hugeicons/react';
import {
  ArrowDown01Icon,
  Attachment01Icon,
  Cancel01Icon,
  File02Icon,
  Mic01Icon,
  PlusSignIcon,
  Tick02Icon
} from '@hugeicons/core-free-icons';

export interface PromptBarSource {
  key: string;
  name: string;
  description?: string;
  icon?: ReactNode | IconSvgElement;
  attach?: boolean;
}

export interface PromptBarCommand {
  key: string;
  name: string;
  description?: string;
  insertText?: string;
  icon?: ReactNode | IconSvgElement;
}

export interface PromptBarModel {
  key: string;
  name: string;
  tag?: string;
}

export interface PromptBarSendDetail {
  attachments: string[];
  model?: PromptBarModel;
  effort?: string;
}

export interface PromptBarLabels {
  promptAriaLabel?: string;
  addAriaLabel?: string;
  commandsButtonLabel?: string;
  commandsAriaLabel?: string;
  sourcesAriaLabel?: string;
  modelsAriaLabel?: string;
  sendAriaLabel?: string;
  stopAriaLabel?: string;
  noMatchesText?: string;
}

export interface PromptBarProps {
  placeholder?: string;
  initialValue?: string;
  maxLength?: number;
  disabled?: boolean;
  sources?: PromptBarSource[];
  commands?: PromptBarCommand[];
  models?: PromptBarModel[];
  defaultModel?: string;
  busy?: boolean;
  hasExternalAttachment?: boolean;
  attachmentSlot?: ReactNode;
  showCommandsButton?: boolean;
  commandsButtonIcon?: ReactNode;
  toolbarHint?: ReactNode;
  labels?: PromptBarLabels;
  onSend?: (text: string, detail: PromptBarSendDetail) => void;
  onStop?: () => void;
  onAttach?: (sourceKey?: string) => string | string[] | void | Promise<string | string[] | void>;
  onDictate?: () => string | void | Promise<string | void>;
  background?: string;
  color?: string;
  menuBackground?: string;
  sendBackground?: string;
  sendColor?: string;
  width?: number | string;
  radius?: number;
  maxRows?: number;
  morphDuration?: number;
  squash?: number;
  tilt?: number;
  pressScale?: number;
  className?: string;
}

type Row = {
  key: string;
  name: string;
  description?: string;
  tag?: string;
  icon?: ReactNode | IconSvgElement;
  attach?: boolean;
  insertText?: string;
};
type Token = { kind: 'at' | 'slash'; query: string; start: number };
type Latest = Pick<PromptBarProps, 'onSend' | 'onStop' | 'onAttach' | 'onDictate'>;

interface SendGlyphProps {
  busy: boolean;
  morphDuration: number;
  squash: number;
  tilt: number;
}

const ARROW_UP = [12, 4.5, 18.5, 11, 14.25, 11, 14.25, 19.5, 9.75, 19.5, 9.75, 11, 5.5, 11];
const SQUARE = [12, 6, 18, 6, 18, 12, 18, 18, 6, 18, 6, 12, 6, 6];
const EASE_IN_OUT: [number, number, number, number] = [0.77, 0, 0.175, 1];
const LINE = 22;

const DEFAULT_SOURCES: PromptBarSource[] = [
  {
    key: 'files',
    name: 'Photos & files',
    description: 'Upload from this device',
    icon: Attachment01Icon,
    attach: true
  }
];
const EMPTY_COMMANDS: PromptBarCommand[] = [];
const EMPTY_MODELS: PromptBarModel[] = [];

const mix = (a: number, b: number, t: number) => a + (b - a) * t;
const pathAt = (a: number[], b: number[], t: number) => {
  let d = '';
  for (let i = 0; i < a.length; i += 2) {
    d += `${i ? 'L' : 'M'}${mix(a[i], b[i], t).toFixed(2)} ${mix(a[i + 1], b[i + 1], t).toFixed(2)}`;
  }
  return `${d}Z`;
};

const parseToken = (draft: string): Token | null => {
  const m = /(^|\s)([@/])([\w-]*)$/.exec(draft);
  if (!m) return null;
  return { kind: m[2] === '@' ? 'at' : 'slash', query: m[3].toLowerCase(), start: m.index + m[1].length };
};

const renderIcon = (icon: ReactNode | IconSvgElement | undefined, size: number) => {
  if (!icon) return null;
  return isValidElement(icon) ? icon : <HugeiconsIcon icon={icon as IconSvgElement} size={size} strokeWidth={1.8} />;
};

function SendGlyph({ busy, morphDuration, squash, tilt }: SendGlyphProps) {
  const reduce = useReducedMotion();
  const svgRef = useRef<SVGSVGElement>(null);
  const pathRef = useRef<SVGPathElement>(null);
  const dir = useRef(busy ? 1 : -1);
  const t = useMotionValue(busy ? 1 : 0);

  useEffect(() => {
    const target = busy ? 1 : 0;
    dir.current = busy ? 1 : -1;
    if (t.get() === target) return undefined;
    const controls = animate(
      t,
      target,
      reduce ? { duration: 0 } : { duration: morphDuration / 1000, ease: EASE_IN_OUT }
    );
    return () => controls.stop();
  }, [busy, morphDuration, reduce, t]);

  useMotionValueEvent(t, 'change', v => {
    pathRef.current?.setAttribute('d', pathAt(ARROW_UP, SQUARE, v));
    const goo = reduce ? 0 : Math.sin(v * Math.PI);
    const sx = 1 - squash * goo;
    if (svgRef.current) {
      svgRef.current.style.transform = goo ? `rotate(${dir.current * tilt * goo}deg) scale(${sx}, ${1 / sx})` : '';
    }
  });

  return (
    <svg
      ref={svgRef}
      className="block h-4 w-4 origin-center"
      viewBox="0 0 24 24"
      aria-hidden="true"
      fill="currentColor"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinejoin="round"
    >
      <path ref={pathRef} d={pathAt(ARROW_UP, SQUARE, t.get())} />
    </svg>
  );
}

const STYLE = `
@keyframes prompt-bar-pop { from { opacity: 0; transform: translateY(4px) scale(0.98); } }
@keyframes prompt-bar-eq { 0%, 100% { transform: scaleY(0.35); } 50% { transform: scaleY(1); } }
`;

const PromptBar: React.FC<PromptBarProps> = ({
  placeholder = 'Ask anything',
  initialValue = '',
  maxLength,
  disabled = false,
  sources = DEFAULT_SOURCES,
  commands = EMPTY_COMMANDS,
  models = EMPTY_MODELS,
  defaultModel = '',
  busy = false,
  hasExternalAttachment = false,
  attachmentSlot,
  showCommandsButton = false,
  commandsButtonIcon,
  toolbarHint,
  labels,
  onSend,
  onStop,
  onAttach,
  onDictate,
  background = 'var(--card)',
  color = 'var(--foreground)',
  menuBackground = 'var(--popover)',
  sendBackground = 'var(--brand-primary)',
  sendColor = 'var(--brand-primary-text)',
  width = '100%',
  radius = 24,
  maxRows = 5,
  morphDuration = 240,
  squash = 0.12,
  tilt = 8,
  pressScale = 0.96,
  className = ''
}) => {
  const inputId = useId();
  const rootRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const highlightRef = useRef<HTMLSpanElement>(null);
  const rowRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const lastOpen = useRef<string | null>(null);
  const dictation = useRef(0);
  const latest = useRef<Latest>({ onSend, onStop, onAttach, onDictate });

  useEffect(() => {
    latest.current = { onSend, onStop, onAttach, onDictate };
  }, [onSend, onStop, onAttach, onDictate]);

  const [draft, setDraft] = useState(initialValue);
  const [attachments, setAttachments] = useState<string[]>([]);
  const [modelKey, setModelKey] = useState(defaultModel);
  const [plusOpen, setPlusOpen] = useState(false);
  const [slashOpen, setSlashOpen] = useState(false);
  const [modelOpen, setModelOpen] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const [active, setActive] = useState(0);
  const [listening, setListening] = useState(false);
  const [pressed, setPressed] = useState(false);

  const model = models.find(m => m.key === modelKey) ?? models[0];
  const token = dismissed ? null : parseToken(draft);
  const tokenKind =
    token?.kind === 'at' && sources.length > 0
      ? 'at'
      : token?.kind === 'slash' && commands.length > 0
        ? 'slash'
        : null;
  const open = plusOpen
    ? 'at'
    : slashOpen
      ? 'slash'
      : (tokenKind ?? (modelOpen ? 'model' : null));
  const query = plusOpen || slashOpen ? '' : (token?.query ?? '');
  const list: Row[] =
    open === 'at'
      ? sources.filter(s => s.name.toLowerCase().includes(query))
      : open === 'slash'
        ? commands.filter(
            c =>
              c.name.replace(/^\//, '').toLowerCase().startsWith(query) ||
              (c.description?.toLowerCase().includes(query) ?? false)
          )
        : open === 'model'
          ? models
          : [];
  const cursor = Math.min(active, Math.max(0, list.length - 1));
  const canSend = !disabled && (draft.trim().length > 0 || attachments.length > 0 || hasExternalAttachment);
  const armed = (busy && Boolean(onStop)) || canSend;

  const focusInput = () => inputRef.current?.focus({ preventScroll: true });
  const closeMenus = () => {
    setPlusOpen(false);
    setSlashOpen(false);
    setModelOpen(false);
  };

  const listLength = list.length;

  useLayoutEffect(() => {
    const highlight = highlightRef.current;
    if (!highlight || !open) return;
    const row = rowRefs.current[cursor];
    if (!row) {
      highlight.style.opacity = '0';
      return;
    }
    const fresh = lastOpen.current !== open;
    lastOpen.current = open;
    if (fresh) highlight.style.transition = 'none';
    highlight.style.top = `${row.offsetTop}px`;
    highlight.style.height = `${row.offsetHeight}px`;
    highlight.style.opacity = '1';
    if (fresh) {
      void highlight.offsetHeight;
      highlight.style.transition = '';
    }
  }, [open, cursor, listLength]);

  useEffect(() => {
    if (!open) lastOpen.current = null;
  }, [open]);

  useEffect(() => {
    if (!plusOpen && !slashOpen && !modelOpen) return undefined;
    const onDown = (e: globalThis.PointerEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) {
        setPlusOpen(false);
        setSlashOpen(false);
        setModelOpen(false);
      }
    };
    document.addEventListener('pointerdown', onDown);
    return () => document.removeEventListener('pointerdown', onDown);
  }, [plusOpen, slashOpen, modelOpen]);

  useLayoutEffect(() => {
    const el = inputRef.current;
    if (!el) return;
    if (typeof CSS !== 'undefined' && typeof CSS.supports === 'function' && CSS.supports('field-sizing', 'content')) {
      return;
    }
    el.style.height = '0px';
    const max = LINE * maxRows;
    el.style.height = `${Math.min(el.scrollHeight, max)}px`;
    el.style.overflowY = el.scrollHeight > max ? 'auto' : 'hidden';
  }, [draft, maxRows]);

  useEffect(
    () => () => {
      dictation.current += 1;
    },
    []
  );

  const pick = (row: Row) => {
    if (open === 'model') {
      setModelKey(row.key);
      setModelOpen(false);
      focusInput();
      return;
    }
    const head = token ? draft.slice(0, token.start) : draft;
    if (row.attach) {
      setDraft(head);
      Promise.resolve(latest.current.onAttach?.(row.key)).then(files => {
        if (!files) return;
        setAttachments(a => [...a, ...(Array.isArray(files) ? files : [files])]);
      });
    } else if (open === 'at') {
      setDraft(`${head}@${row.name} `);
    } else {
      const insertion = row.insertText ?? `${row.name} `;
      const trimmedHead = head.trimEnd();
      const nextDraft = trimmedHead ? `${trimmedHead} ${insertion}` : insertion;
      setDraft(maxLength ? nextDraft.slice(0, maxLength) : nextDraft);
    }
    setPlusOpen(false);
    setSlashOpen(false);
    setDismissed(false);
    focusInput();
  };

  const send = () => {
    if (!canSend || busy) return;
    latest.current.onSend?.(draft.trim(), { attachments, model });
    setDraft('');
    setAttachments([]);
    setDismissed(false);
    closeMenus();
    focusInput();
  };

  const toggleListen = () => {
    if (listening) {
      dictation.current += 1;
      setListening(false);
      return;
    }
    const seq = ++dictation.current;
    setListening(true);
    Promise.resolve(latest.current.onDictate?.()).then(
      text => {
        if (seq !== dictation.current) return;
        setListening(false);
        if (text) setDraft(d => (d.trim() ? `${d.trimEnd()} ${text}` : text));
        focusInput();
      },
      () => {
        if (seq === dictation.current) setListening(false);
      }
    );
  };

  const onKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (open && list.length) {
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        e.preventDefault();
        setActive((cursor + (e.key === 'ArrowDown' ? 1 : list.length - 1)) % list.length);
        return;
      }
      if ((e.key === 'Enter' && !e.shiftKey) || e.key === 'Tab') {
        if (e.nativeEvent.isComposing || e.keyCode === 229) {
          return;
        }
        e.preventDefault();
        pick(list[cursor]);
        return;
      }
    }
    if (e.key === 'Escape') {
      if (open) {
        e.preventDefault();
        setDismissed(true);
        closeMenus();
      }
      return;
    }
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (e.nativeEvent.isComposing || e.keyCode === 229) {
        return;
      }
      send();
    }
  };

  const down = (e: PointerEvent<HTMLButtonElement>) => {
    if (e.button !== 0 || !armed) return;
    setPressed(true);
  };
  const up = () => setPressed(false);

  const widthCss = typeof width === 'number' ? `${width}px` : width;
  const maxHeightPx = LINE * maxRows;

  return (
    <div
      ref={rootRef}
      className={`group relative text-[14px] leading-[22px] [width:min(var(--pb-w),100%)] [color:var(--pb-ink)]${className ? ` ${className}` : ''}`}
      data-busy={busy ? '' : undefined}
      style={
        {
          '--pb-bg': background,
          '--pb-ink': color,
          '--pb-menu': menuBackground,
          '--pb-send-bg': sendBackground,
          '--pb-send-ink': sendColor,
          '--pb-w': widthCss,
          '--pb-radius': `${radius}px`,
          '--pb-press': pressScale
        } as CSSProperties
      }
    >
      <style>{STYLE}</style>
      {open ? (
        <div
          className="absolute inset-x-0 bottom-[calc(100%+8px)] z-20 origin-bottom rounded-2xl border border-border p-1.5 shadow-none [animation:prompt-bar-pop_180ms_cubic-bezier(0.23,1,0.32,1)_both] [background:var(--pb-menu)] data-[kind=model]:right-auto data-[kind=model]:w-[200px] data-[kind=model]:origin-bottom-left motion-reduce:[animation:none]"
          role="listbox"
          aria-label={
            open === 'at'
              ? (labels?.sourcesAriaLabel ?? 'Sources')
              : open === 'slash'
                ? (labels?.commandsAriaLabel ?? 'Commands')
                : (labels?.modelsAriaLabel ?? 'Models')
          }
          data-kind={open}
        >
          <span
            ref={highlightRef}
            className="pointer-events-none absolute inset-x-1.5 rounded-xl opacity-0 [background:color-mix(in_srgb,var(--pb-ink)_8%,transparent)] [transition:top_220ms_cubic-bezier(0.23,1,0.32,1),height_220ms_cubic-bezier(0.23,1,0.32,1),opacity_150ms_ease] motion-reduce:[transition:opacity_150ms_ease]"
            aria-hidden="true"
          />
          {list.map((row, i) => (
            <button
              key={row.key}
              ref={el => {
                rowRefs.current[i] = el;
              }}
              type="button"
              role="option"
              aria-selected={i === cursor}
              className="relative z-[1] flex min-h-10 w-full cursor-pointer items-center gap-2.5 rounded-xl border-0 bg-transparent px-2.5 py-2 text-left text-inherit outline-none [font:inherit] [-webkit-tap-highlight-color:transparent]"
              onMouseDown={e => e.preventDefault()}
              onPointerEnter={() => setActive(i)}
              onClick={() => pick(row)}
            >
              {row.icon ? (
                <span className="inline-flex w-5 flex-none justify-center [color:color-mix(in_srgb,var(--pb-ink)_70%,transparent)]">
                  {renderIcon(row.icon, 16)}
                </span>
              ) : null}
              <span className="flex-none text-[13px] font-medium">{row.name}</span>
              {row.description ? (
                <span className="min-w-0 flex-auto truncate text-[12px] [color:color-mix(in_srgb,var(--pb-ink)_55%,transparent)]">
                  {row.description}
                </span>
              ) : null}
              {open === 'model' ? (
                <>
                  <span className="ml-auto flex-none text-[11px] [color:color-mix(in_srgb,var(--pb-ink)_55%,transparent)]">
                    {row.tag}
                  </span>
                  <span
                    className="inline-flex w-4 flex-none justify-center opacity-0 data-[on]:opacity-100"
                    data-on={row.key === model?.key ? '' : undefined}
                  >
                    <HugeiconsIcon icon={Tick02Icon} size={13} strokeWidth={2.5} />
                  </span>
                </>
              ) : null}
            </button>
          ))}
          {list.length === 0 ? (
            <div className="flex h-9 items-center px-2.5 text-[12px] [color:color-mix(in_srgb,var(--pb-ink)_55%,transparent)]">
              {(labels?.noMatchesText ?? 'No matches for "{query}"').replace('{query}', query)}
            </div>
          ) : null}
        </div>
      ) : null}

      <div
        className="relative isolate flex cursor-text flex-col gap-2.5 border border-border px-3.5 pt-3.5 pb-2.5 shadow-none transition-colors duration-150 focus-within:border-[var(--border-strong)] [background:var(--pb-bg)] [border-radius:var(--pb-radius)]"
        role="presentation"
        onPointerDown={e => {
          if (e.target === e.currentTarget || e.target === inputRef.current) closeMenus();
        }}
        onClick={focusInput}
      >
        {attachmentSlot}

        {attachments.length > 0 ? (
          <div className="flex flex-wrap gap-1.5">
            {attachments.map((file, i) => (
              <span
                key={`${file}-${i}`}
                className="inline-flex h-[26px] items-center gap-1.5 rounded-xl pr-1 pl-2 text-[12px] [animation:prompt-bar-pop_200ms_cubic-bezier(0.23,1,0.32,1)_both] [background:color-mix(in_srgb,var(--pb-ink)_8%,transparent)] motion-reduce:[animation:none]"
              >
                <HugeiconsIcon icon={File02Icon} size={12} strokeWidth={2} />
                <span className="max-w-[144px] truncate">{file}</span>
                <button
                  type="button"
                  className="inline-grid h-[18px] w-[18px] cursor-pointer place-items-center rounded-full border-0 bg-transparent p-0 text-inherit opacity-60 outline-none [transition:opacity_120ms_ease,background-color_120ms_ease] hover:opacity-100 hover:[background:color-mix(in_srgb,var(--pb-ink)_10%,transparent)]"
                  aria-label={`Remove ${file}`}
                  onClick={() => setAttachments(a => a.filter((_, j) => j !== i))}
                >
                  <HugeiconsIcon icon={Cancel01Icon} size={10} strokeWidth={2.5} />
                </button>
              </span>
            ))}
          </div>
        ) : null}

        <label htmlFor={inputId} className="sr-only">
          {labels?.promptAriaLabel ?? placeholder}
        </label>
        <textarea
          id={inputId}
          ref={inputRef}
          className="block h-auto min-h-[24px] w-full resize-none border-0 bg-transparent px-0.5 py-0 text-[14px] leading-[22px] text-inherit outline-none [field-sizing:content] [font:inherit] [overflow-wrap:anywhere] placeholder:[color:color-mix(in_srgb,var(--pb-ink)_48%,transparent)] disabled:opacity-60 [@media(pointer:coarse)]:text-[16px]"
          style={{ maxHeight: `${maxHeightPx}px` }}
          rows={1}
          maxLength={maxLength}
          disabled={disabled}
          value={draft}
          placeholder={listening ? 'Listening…' : placeholder}
          onChange={e => {
            setDraft(e.target.value);
            setDismissed(false);
            closeMenus();
            setActive(0);
          }}
          onFocus={closeMenus}
          onKeyDown={onKeyDown}
        />

        <div className="flex items-center gap-1.5">
          {sources.length > 0 ? (
            <button
              type="button"
              disabled={disabled}
              className="inline-grid size-9 flex-none cursor-pointer touch-manipulation place-items-center rounded-full border border-border bg-transparent p-0 outline-none select-none [color:color-mix(in_srgb,var(--pb-ink)_72%,transparent)] [font:inherit] [-webkit-tap-highlight-color:transparent] [transition:background-color_150ms_ease,color_150ms_ease,transform_160ms_cubic-bezier(0.23,1,0.32,1)] active:[transform:scale(0.94)] focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)] focus-visible:ring-offset-2 focus-visible:ring-offset-card disabled:cursor-not-allowed disabled:opacity-50 data-[on]:[background:color-mix(in_srgb,var(--pb-ink)_8%,transparent)] data-[on]:[color:var(--pb-ink)] motion-reduce:active:[transform:none] [@media(hover:hover)_and_(pointer:fine)]:hover:[background:color-mix(in_srgb,var(--pb-ink)_8%,transparent)] [@media(hover:hover)_and_(pointer:fine)]:hover:[color:var(--pb-ink)]"
              aria-label={labels?.addAriaLabel ?? 'Add files and sources'}
              aria-expanded={plusOpen}
              data-on={plusOpen ? '' : undefined}
              onMouseDown={e => e.preventDefault()}
              onClick={() => {
                setSlashOpen(false);
                setModelOpen(false);
                setActive(0);
                setPlusOpen(v => !v);
                focusInput();
              }}
            >
              <HugeiconsIcon icon={PlusSignIcon} size={16} strokeWidth={2} />
            </button>
          ) : null}

          {showCommandsButton && commands.length > 0 ? (
            <button
              type="button"
              disabled={disabled}
              className="inline-flex h-9 flex-none cursor-pointer touch-manipulation items-center gap-1.5 rounded-full border border-transparent bg-transparent px-3 text-xs font-medium outline-none select-none [color:color-mix(in_srgb,var(--pb-ink)_68%,transparent)] [font:inherit] [-webkit-tap-highlight-color:transparent] [transition:background-color_150ms_ease,color_150ms_ease] focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)] focus-visible:ring-offset-2 focus-visible:ring-offset-card disabled:cursor-not-allowed disabled:opacity-50 data-[on]:[background:color-mix(in_srgb,var(--pb-ink)_8%,transparent)] data-[on]:[color:var(--pb-ink)] [@media(hover:hover)_and_(pointer:fine)]:hover:[background:color-mix(in_srgb,var(--pb-ink)_8%,transparent)] [@media(hover:hover)_and_(pointer:fine)]:hover:[color:var(--pb-ink)]"
              aria-label={labels?.commandsAriaLabel ?? 'Prompt shortcuts'}
              aria-expanded={slashOpen}
              data-on={slashOpen ? '' : undefined}
              onMouseDown={e => e.preventDefault()}
              onClick={() => {
                setPlusOpen(false);
                setModelOpen(false);
                setActive(0);
                setSlashOpen(v => !v);
                focusInput();
              }}
            >
              {commandsButtonIcon}
              <span>{labels?.commandsButtonLabel ?? 'Templates'}</span>
            </button>
          ) : null}

          {models.length > 0 ? (
            <button
              type="button"
              disabled={disabled}
              className="inline-flex h-9 flex-none cursor-pointer touch-manipulation items-center gap-1 rounded-full border-0 bg-transparent px-2.5 text-[12px] font-medium outline-none select-none [color:color-mix(in_srgb,var(--pb-ink)_70%,transparent)] [font:inherit] [-webkit-tap-highlight-color:transparent] [transition:background-color_150ms_ease,color_150ms_ease] data-[on]:[background:color-mix(in_srgb,var(--pb-ink)_8%,transparent)] data-[on]:[color:var(--pb-ink)] [@media(hover:hover)_and_(pointer:fine)]:hover:[background:color-mix(in_srgb,var(--pb-ink)_8%,transparent)] [@media(hover:hover)_and_(pointer:fine)]:hover:[color:var(--pb-ink)]"
              aria-label={labels?.modelsAriaLabel ?? 'Choose model'}
              aria-expanded={modelOpen}
              data-on={modelOpen ? '' : undefined}
              onMouseDown={e => e.preventDefault()}
              onClick={() => {
                setPlusOpen(false);
                setSlashOpen(false);
                setActive(Math.max(0, models.indexOf(model)));
                setModelOpen(v => !v);
                focusInput();
              }}
            >
              <span>{model.name}</span>
              <HugeiconsIcon icon={ArrowDown01Icon} size={12} strokeWidth={2.4} />
            </button>
          ) : null}

          <span className="flex-auto" />

          {toolbarHint}

          {onDictate ? (
            <button
              type="button"
              disabled={disabled}
              className="inline-grid size-9 flex-none cursor-pointer touch-manipulation place-items-center rounded-full border-0 bg-transparent p-0 outline-none select-none [color:color-mix(in_srgb,var(--pb-ink)_60%,transparent)] [font:inherit] [-webkit-tap-highlight-color:transparent] [transition:background-color_150ms_ease,color_150ms_ease,transform_160ms_cubic-bezier(0.23,1,0.32,1)] active:[transform:scale(0.94)] data-[on]:[background:color-mix(in_srgb,var(--pb-ink)_8%,transparent)] data-[on]:[color:var(--pb-ink)] motion-reduce:active:[transform:none] [@media(hover:hover)_and_(pointer:fine)]:hover:[background:color-mix(in_srgb,var(--pb-ink)_8%,transparent)] [@media(hover:hover)_and_(pointer:fine)]:hover:[color:var(--pb-ink)]"
              aria-label={listening ? 'Stop dictation' : 'Dictate'}
              aria-pressed={listening}
              data-on={listening ? '' : undefined}
              onMouseDown={e => e.preventDefault()}
              onClick={toggleListen}
            >
              {listening ? (
                <span
                  className="flex h-3.5 items-center gap-[2.5px] [&>i]:block [&>i]:h-full [&>i]:w-[2.5px] [&>i]:origin-center [&>i]:rounded-full [&>i]:bg-current [&>i]:[animation:prompt-bar-eq_900ms_ease-in-out_infinite] [&>i:nth-child(2)]:[animation-delay:150ms] [&>i:nth-child(3)]:[animation-delay:300ms]"
                  aria-hidden="true"
                >
                  <i />
                  <i />
                  <i />
                </span>
              ) : (
                <HugeiconsIcon icon={Mic01Icon} size={15} strokeWidth={2} />
              )}
            </button>
          ) : null}

          <button
            type="button"
            className="relative inline-grid size-9 flex-none cursor-pointer touch-manipulation place-items-center rounded-full border-0 p-0 outline-none select-none [background:color-mix(in_srgb,var(--pb-ink)_12%,var(--pb-bg))] [color:color-mix(in_srgb,var(--pb-ink)_48%,var(--pb-bg))] [font:inherit] [-webkit-tap-highlight-color:transparent] [transition:background-color_200ms_ease,color_200ms_ease,transform_160ms_cubic-bezier(0.23,1,0.32,1)] focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)] focus-visible:ring-offset-2 focus-visible:ring-offset-card disabled:cursor-not-allowed data-[armed]:[background:var(--pb-send-bg)] data-[armed]:[color:var(--pb-send-ink)] data-[pressed]:[transform:scale(var(--pb-press))] motion-reduce:data-[pressed]:[transform:none]"
            disabled={!armed}
            aria-label={busy ? (labels?.stopAriaLabel ?? 'Stop') : (labels?.sendAriaLabel ?? 'Send')}
            title={busy ? (labels?.stopAriaLabel ?? 'Stop') : (labels?.sendAriaLabel ?? 'Send')}
            data-armed={armed ? '' : undefined}
            data-pressed={pressed ? '' : undefined}
            onMouseDown={e => e.preventDefault()}
            onPointerDown={down}
            onPointerUp={up}
            onPointerCancel={up}
            onPointerLeave={up}
            onClick={() => {
              if (busy) latest.current.onStop?.();
              else send();
            }}
          >
            <SendGlyph busy={busy} morphDuration={morphDuration} squash={squash} tilt={tilt} />
          </button>
        </div>
      </div>
    </div>
  );
};

export default PromptBar;
