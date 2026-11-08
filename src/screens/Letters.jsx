import React, { useEffect, useMemo, useRef, useState } from 'react';
import { ArrowLeft, Mail } from 'lucide-react';
import { BackButton } from '../components/UI';

const dateFormat = new Intl.DateTimeFormat('fa-IR', { dateStyle: 'long' });
const dayFormat = new Intl.DateTimeFormat('fa-IR', { day: 'numeric', month: 'long' });
const formatLong = (value) => {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? '' : dateFormat.format(date);
};
const formatShort = (value) => {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return date.getFullYear() === new Date().getFullYear() ? dayFormat.format(date) : dateFormat.format(date);
};

/** One letter, alone on the page. Nothing else competes with the words. */
function LetterReader({ letter, onClose, onDiscuss }) {
  const articleRef = useRef(null);
  const progressRef = useRef(null);

  useEffect(() => { articleRef.current?.focus({ preventScroll: true }); }, [letter.id]);
  useEffect(() => {
    const onKey = (event) => { if (event.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  const onScroll = (event) => {
    const node = event.currentTarget;
    const range = node.scrollHeight - node.clientHeight;
    if (progressRef.current) progressRef.current.style.transform = `scaleX(${range > 0 ? Math.min(1, node.scrollTop / range) : 0})`;
  };

  let step = 0;
  const reveal = () => ({ '--i': step++ });

  return (
    <section className="app-screen theme-dark letters-screen letters-screen--reading">
      <BackButton dark onClick={onClose} label="بازگشت به نامه‌ها" />
      <div className="letter-progress" aria-hidden="true"><i ref={progressRef} /></div>
      <div className="letters-scroll" onScroll={onScroll}>
        <article className="letter-read" ref={articleRef} tabIndex={-1} aria-label={letter.title}>
          <time className="letter-reveal" style={reveal()} dateTime={letter.createdAt}>{formatLong(letter.createdAt)}</time>
          <h1 className="letter-read-title letter-reveal" style={reveal()}>{letter.title}</h1>
          <p className="letter-read-salutation letter-reveal" style={reveal()}>{letter.salutation || 'دوست من،'}</p>
          {letter.paragraphs.map((paragraph, index) => (
            <p className="letter-read-body letter-reveal" style={reveal()} key={index}>{paragraph}</p>
          ))}
          {letter.question && <p className="letter-read-question letter-reveal" style={reveal()}>{letter.question}</p>}
          <p className="letter-read-signature letter-reveal" style={reveal()}>کنار</p>

          <footer className="letter-end letter-reveal" style={reveal()}>
            {onDiscuss && (
              <button type="button" className="letter-discuss" onClick={() => onDiscuss(letter)}>
                <span>دربارهٔ این نامه حرف بزنیم</span>
                <ArrowLeft size={17} strokeWidth={1.6} aria-hidden="true" />
              </button>
            )}

          </footer>
        </article>
      </div>
    </section>
  );
}

/** A quiet inbox: unread letters glow softly, read ones fade back. */
function Inbox({ back, phase, letters, readIds, onOpen, onReload, focusId }) {
  const listRef = useRef(null);
  useEffect(() => {
    if (!focusId) return;
    listRef.current?.querySelector(`[data-letter-id="${CSS.escape(focusId)}"]`)?.focus({ preventScroll: true });
  }, [focusId]);

  let body;
  if (phase === 'loading' || phase === 'idle') {
    body = (
      <div className="letters-ghosts" role="status">
        <span className="sr-only">در حال بارگذاری نامه‌ها…</span>
        {[0, 1, 2].map((n) => <i key={n} style={{ '--i': n }} aria-hidden="true" />)}
      </div>
    );
  } else if (phase === 'error') {
    body = (
      <div className="letters-empty" role="alert">
        <p>نامه‌ها بارگذاری نشدن.</p>
        <button type="button" onClick={onReload}>تلاش دوباره</button>
      </div>
    );
  } else if (letters.length === 0) {
    body = (
      <div className="letters-empty">
        <Mail size={34} strokeWidth={1.1} aria-hidden="true" />
        <p>هنوز نامه‌ای نرسیده.</p>
      </div>
    );
  } else {
    body = (
      <ol className="letters-list" ref={listRef}>
        {letters.map((letter, index) => {
          const unread = !readIds.includes(letter.id);
          return (
            <li key={letter.id}>
              <button
                type="button"
                data-letter-id={letter.id}
                className={`letter-row ${unread ? 'is-unread' : ''}`}
                style={{ '--i': index }}
                onClick={() => onOpen(letter.id)}
              >
                <span className="letter-mark" aria-hidden="true" />
                <span className="letter-row-text">
                  <span className="letter-row-title">{letter.title}</span>
                  <time dateTime={letter.createdAt}>{formatShort(letter.createdAt)}</time>
                </span>
                {unread && <span className="sr-only">نخوانده</span>}
              </button>
            </li>
          );
        })}
      </ol>
    );
  }

  return (
    <section className="app-screen theme-dark letters-screen">
      <BackButton dark onClick={back} />
      <div className="letters-scroll">
        <header className="letters-head"><h1>نامه‌های کنار</h1></header>
        {body}
      </div>
    </section>
  );
}

export default function Letters({
  back, letterPhase = 'idle', letters = [], readLetterIds = [], onReadLetter, onReloadLetters,
  onDiscussLetter,
}) {
  const [openId, setOpenId] = useState(null);
  const [lastOpened, setLastOpened] = useState(null);
  const ordered = useMemo(
    () => (letterPhase === 'ready' ? [...letters].sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt)) : []),
    [letterPhase, letters],
  );
  const open = ordered.find((item) => item.id === openId) || null;

  // A letter counts as read the moment it is opened, never while it sits in the list.
  useEffect(() => { if (open) onReadLetter?.(open.id); }, [open?.id, onReadLetter]);

  const close = () => { setLastOpened(openId); setOpenId(null); };

  if (open) {
    return (
      <LetterReader
        key={open.id}
        letter={open}
        onClose={close}
        onDiscuss={onDiscussLetter}
      />
    );
  }
  return (
    <Inbox
      back={back}
      phase={letterPhase === 'unavailable' ? 'ready' : letterPhase}
      letters={ordered}
      readIds={readLetterIds}
      onOpen={setOpenId}
      onReload={onReloadLetters}
      focusId={lastOpened}
    />
  );
}
