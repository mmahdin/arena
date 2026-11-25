/* Progressive enhancement only. The server still validates phone, OTP and resend limits. */
(() => {
  'use strict';
  const digits = (value) => String(value || '').replace(/[\u06f0-\u06f9\u0660-\u0669]/g, (char) => {
    const code = char.charCodeAt(0);
    return String(code >= 0x06f0 ? code - 0x06f0 : code - 0x0660);
  });
  const error = document.getElementById('theia-auth-error');
  const dismissError = (input) => {
    if (error) error.hidden = true;
    input?.removeAttribute('aria-invalid');
  };

  const phoneForm = document.getElementById('theia-phone-form');
  const phone = document.getElementById('theia-phone');
  if (phoneForm && phone) {
    phone.addEventListener('input', () => dismissError(phone));
    phoneForm.addEventListener('submit', () => {
      phone.value = digits(phone.value).trim();
      if (!phone.checkValidity()) return;
      const button = phoneForm.querySelector('button[type="submit"]');
      if (button) button.disabled = true;
    });
  }

  const codeForm = document.getElementById('theia-code-form');
  const code = document.getElementById('theia-code');
  const otp = document.getElementById('theia-otp');
  if (codeForm && code && otp) {
    const cells = Array.from(otp.querySelectorAll('.theia-otp__cell'));
    if (cells.length === 6) {
      const refresh = () => {
        const normalized = digits(code.value).replace(/[^0-9]/g, '').slice(0, 6);
        if (code.value !== normalized) code.value = normalized;
        cells.forEach((cell, i) => {
          cell.textContent = normalized.charAt(i);
          cell.classList.toggle('is-filled', i < normalized.length);
          cell.classList.toggle('is-current', i === Math.min(normalized.length, 5));
        });
      };
      code.addEventListener('input', () => { dismissError(code); refresh(); });
      code.addEventListener('change', refresh);
      code.addEventListener('focus', refresh);
      otp.addEventListener('click', () => code.focus());
      codeForm.addEventListener('submit', () => {
        refresh();
        if (!code.checkValidity()) return;
        const button = codeForm.querySelector('button[type="submit"]');
        if (button) button.disabled = true;
      });
      refresh();
      otp.classList.add('is-enhanced');
    }
  }

  const resend = document.getElementById('theia-resend-button');
  const timer = document.getElementById('theia-resend-timer');
  const ready = document.getElementById('theia-resend-ready');
  if (resend && timer) {
    const configured = Number.parseInt(resend.dataset.waitSeconds || '0', 10);
    const initial = Number.isFinite(configured) ? Math.min(60, Math.max(0, configured)) : 0;
    // An absolute deadline survives inactive tabs and browser timer throttling.
    const deadline = Date.now() + initial * 1000;
    let remaining = initial;
    let interval = null;
    const fa = (number) => String(number).padStart(2, '0').replace(/[0-9]/g, (char) =>
      String.fromCharCode(0x06f0 + Number(char)));
    const render = () => {
      const previous = remaining;
      remaining = Math.max(0, Math.ceil((deadline - Date.now()) / 1000));
      resend.disabled = remaining > 0;
      timer.textContent = remaining > 0 ? `${fa(Math.floor(remaining / 60))}:${fa(remaining % 60)}` : '';
      if (remaining === 0 && previous > 0 && ready) {
        ready.textContent = 'حالا می‌تونی دوباره کد بگیری.';
      }
      if (remaining === 0 && interval !== null) {
        window.clearInterval(interval);
        interval = null;
      }
    };
    render();
    if (initial > 0) interval = window.setInterval(render, 1000);
    document.addEventListener('visibilitychange', render);
    window.addEventListener('pageshow', render);
  }

  /* Clear disabled submit controls when returning from the browser back/forward cache. */
  window.addEventListener('pageshow', () => {
    document.querySelectorAll('.theia-primary:disabled').forEach((button) => { button.disabled = false; });
  });
})();
