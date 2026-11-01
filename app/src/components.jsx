import React from 'react';
import { ArrowBack, Check, Logo, Mic, Info, Search, Spark, Wave, PathArt } from './icons.jsx';

export const TopBar = ({ onBack, prog = 0 }) => (
  <>
    <div className="topbar">
      <button className="icon-btn" onClick={onBack} aria-label="بازگشت"><ArrowBack size={22} /></button>
      <Logo />
      <span />
    </div>
    {prog > 0 && (
      <div className="progress">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className={`seg ${i <= prog ? 'filled' : ''}`}>
            <span className={`dot ${i < prog ? 'done' : i === prog ? 'current' : ''}`}>
              {i < prog && <Check size={9} />}
            </span>
          </div>
        ))}
      </div>
    )}
  </>
);

export const OptionList = ({ options, value, onChange, notes }) => (
  <div className="option-list">
    {options.map((opt) => (
      <button
        key={opt}
        className={`option-row ${value === opt ? 'selected' : ''}`}
        onClick={() => onChange(opt)}
      >
        <span>{opt}</span>
        {notes && notes[opt] && <span className="row-note">{notes[opt]}</span>}
        <span className="radio" />
      </button>
    ))}
  </div>
);

export const Foot = ({ children }) => <div className="footer-actions">{children}</div>;

export const TextareaField = ({ value, onChange, placeholder, words, suggestion, onMic }) => (
  <>
    <div className="textarea-wrap">
      <textarea
        className="textarea"
        value={value}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)}
      />
      <button className="mic-fab" onClick={onMic} aria-label="میکروفون"><Mic size={18} /></button>
    </div>
    <div className="word-count">
      <span>{words} کلمه</span>
      <span className="bar"><i style={{ width: `${Math.min(100, words * 4)}%` }} /></span>
    </div>
    {suggestion && (
      <div className="suggestion"><Spark size={14} /> {suggestion}</div>
    )}
  </>
);

export const InfoCard = ({ children }) => (
  <div className="info-card"><Info /> <span>{children}</span></div>
);

export const DarkCardScreen = ({ card, onNext }) => (
  <div className="screen dark">
    <PathArt />
    <div className="onb-body">
      <div className="eyebrow">{card.eyebrow}</div>
      <h1 className="display onb-title">{card.title}</h1>
      <p className="onb-text">{card.body}</p>
      {card.warn && (
        <div className="onb-warn"><Info /> <span>{card.warn}</span></div>
      )}
      <button className="btn btn-white" onClick={onNext}>{card.cta}</button>
    </div>
  </div>
);

export const StatScreen = ({ variant, num, text, hl, quote, source, onNext }) => (
  <div className={`screen stat-screen`}>
    <div className={`stat-bg bg-${variant}`} />
    <div className="stat-inner">
      <div className="stat-num display">{num}</div>
      <p className="stat-text">{text} {hl && <span className="hl display">«{hl}»</span>}</p>
      <div className="quote-card">
        <p className="q">«{quote}»</p>
        <div className="src">{source}</div>
      </div>
      <button className="btn btn-white" onClick={onNext}>ادامه</button>
    </div>
  </div>
);

export const GradCardScreen = ({ variant, title, sub, onNext }) => (
  <div className={`screen grad-card-screen`}>
    <div className={`stat-bg bg-${variant}`} />
    <div className="inner">
      <h2 className="display">{title}</h2>
      <p>{sub}</p>
      <button className="btn btn-white" onClick={onNext}>ادامه</button>
    </div>
  </div>
);

export const ReflectionScreen = ({ eyebrow, text, onNext, onBack }) => (
  <div className="screen light">
    <TopBar onBack={onBack || onNext} prog={2} />
    <div className="reflection">
      <span className="bubble">
        <svg width="54" height="54" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round">
          <path d="M21 12a8 8 0 0 1-8 8H4l2.2-2.6A8 8 0 1 1 21 12z" />
          <path d="M9 12l2 2 4-4" />
        </svg>
      </span>
      <div className="eyebrow">{eyebrow}</div>
      <p className="display">{text}</p>
    </div>
    <Foot><button className="btn btn-black" onClick={onNext}>ادامه</button></Foot>
  </div>
);

export const SearchBox = ({ value, onChange, placeholder }) => (
  <div className="search-wrap">
    <Search />
    <input className="search-input" value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder} />
  </div>
);
