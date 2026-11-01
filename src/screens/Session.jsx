import React, { useEffect, useRef, useState } from 'react';
import { SESSION_FIRST, SESSION_REPLY } from '../data.js';
import { faNum } from '../components/UI.jsx';
import { X, Mic, Dots, Clock, Waveform } from '../components/Icons.jsx';

export function Session({ guide, name, duration, onClose, onPasteToast }) {
  const [msgs, setMsgs] = useState([
    { role: 'guide', first: true, text: SESSION_FIRST(name || 'دوست عزیز') },
  ]);
  const [text, setText] = useState('');
  const [thinking, setThinking] = useState(false);
  const [tip, setTip] = useState(true);
  const [left, setLeft] = useState(duration);
  const logRef = useRef(null);

  useEffect(() => {
    const t = setInterval(() => setLeft((l) => Math.max(0, l - 1)), 60000);
    return () => clearInterval(t);
  }, []);

  useEffect(() => {
    logRef.current?.scrollTo({ top: 9e5, behavior: 'smooth' });
  }, [msgs, thinking]);

  const send = () => {
    const t = text.trim();
    if (!t || thinking) return;
    setMsgs((m) => [...m, { role: 'user', text: t }]);
    setText('');
    setThinking(true);
    setTimeout(() => {
      setThinking(false);
      setMsgs((m) => [...m, { role: 'guide', text: SESSION_REPLY }]);
    }, 2200);
  };

  return (
    <div className="screen sess pop">
      {msgs.length <= 1 ? (
        <div className="hero">
          <img src={guide.img} alt={guide.name} />
          <div className="fade" />
          <button className="close" type="button" onClick={onClose} aria-label="بستن"
            style={{ position: 'absolute', top: 20, insetInlineEnd: 20, width: 46, height: 46, borderRadius: '50%', background: 'rgba(90,90,90,.4)', border: 'none', color: '#fff', display: 'grid', placeItems: 'center', cursor: 'pointer', zIndex: 4 }}>
            <X size={20} />
          </button>
          <div className="meta">
            <div className="eyebrow-w" style={{ color: '#d8d8d4', fontSize: 14, fontWeight: 700, letterSpacing: '.16em' }}>راهنمای تو</div>
            <h2 style={{ fontFamily: 'var(--f-display)', fontWeight: 700, fontSize: 34, margin: '6px 0 0', color: '#fff' }}>{guide.name}</h2>
            <div className="tags" style={{ display: 'flex', gap: 10, marginTop: 14 }}>
              {guide.tags.map((t) => (
                <span key={t} className="gtag" style={{ background: 'rgba(30,30,28,.35)', border: '1px solid rgba(255,255,255,.55)', borderRadius: 999, padding: '10px 22px' }}>{t}</span>
              ))}
            </div>
          </div>
        </div>
      ) : (
        <div className="hdr">
          <img src={guide.img} alt={guide.name} />
          <div>
            <div className="nm">{guide.name}</div>
            <div className="left"><Clock size={16} /> {faNum(left)} دقیقه مانده</div>
          </div>
          <button className="dots" type="button" aria-label="گزینه‌ها"><Dots /></button>
        </div>
      )}

      <div className="log" ref={logRef}>
        {msgs.map((m, i) =>
          m.role === 'guide' ? (
            <div className="msg-g" key={i}>
              {m.first && <h3 className="welcome">به اولین جلسه‌ات خوش آمدی.</h3>}
              <p>{m.text}</p>
            </div>
          ) : (
            <div className="msg-u" key={i}>{m.text}</div>
          ),
        )}
        {thinking && (
          <div className="thinking">
            <img src={guide.img} alt="" />
            یک دقیقه وقت می‌خواهم…
          </div>
        )}
      </div>

      <div className="composer">
        {tip && (
          <div className="tip" onClick={() => setTip(false)}>برای حالت صوتی ضربه بزن</div>
        )}
        <div className="bar">
          <input
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && send()}
            onPaste={() => onPasteToast()}
            placeholder={thinking ? 'راهنمایت در حال فکر کردن است…' : 'اینجا بنویس…'}
          />
          <button className="micbtn" type="button" disabled={thinking} onClick={() => setTip(false)} aria-label="صدا">
            <Mic size={21} />
          </button>
        </div>
      </div>
    </div>
  );
}
