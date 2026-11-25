<#ftl output_format="HTML" auto_esc=true>
<#import "theia-layout.ftl" as layout>
<@layout.page title="تأیید شماره موبایل" step="code">
  <div class="theia-navigation">
    <form method="post" action="${url.loginAction}" class="theia-back-form">
      <input type="hidden" name="smsAction" value="change" />
      <button type="submit" class="theia-back" aria-label="تغییر شماره موبایل" title="تغییر شماره موبایل">
        <svg width="19" height="19" viewBox="0 0 20 20" fill="none" aria-hidden="true"><path d="m12.5 4.5-5.5 5.5 5.5 5.5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg>
      </button>
    </form>
  </div>
  <section class="theia-intro theia-intro--code" aria-labelledby="theia-title">
    <span class="theia-eyebrow">یک قدم تا ورود</span>
    <h1 id="theia-title">تأیید شماره موبایل</h1>
    <p>کد ۶ رقمی ارسال‌شده به <b class="theia-phone-number" dir="ltr">${(smsMaskedPhone!'')}</b> رو وارد کن.</p>
  </section>

  <#if smsError?? && smsError?has_content>
    <div id="theia-auth-error" class="theia-error" role="alert">
      <span class="theia-error__icon" aria-hidden="true">!</span>
      <span>${smsError}</span>
    </div>
  </#if>

  <form id="theia-code-form" class="theia-form theia-code-form" action="${url.loginAction}" method="post" accept-charset="UTF-8">
    <input type="hidden" name="smsAction" value="verify" />
    <div class="theia-field">
      <label id="theia-code-label" for="theia-code">کد تأیید</label>
      <div class="theia-otp" id="theia-otp">
        <input id="theia-code" class="theia-code-input" type="text" name="code" dir="ltr"
               inputmode="numeric" autocomplete="one-time-code" enterkeyhint="done"
               spellcheck="false" autocapitalize="off" maxlength="6"
               pattern="[0-9۰-۹٠-٩]{6}" required
               aria-labelledby="theia-code-label" aria-describedby="theia-code-help<#if smsError?? && smsError?has_content> theia-auth-error</#if>"
               <#if smsError?? && smsError?has_content>aria-invalid="true"</#if> />
        <div class="theia-otp__cells" aria-hidden="true">
          <#list 1..6 as i><span class="theia-otp__cell"></span></#list>
        </div>
      </div>
      <p id="theia-code-help" class="theia-field__hint">کد تا ۲ دقیقه معتبره. می‌تونی کد رو اینجا جای‌گذاری کنی.</p>
    </div>
    <button class="theia-primary" type="submit">تأیید و ادامه <span class="theia-primary__arrow" aria-hidden="true">←</span></button>
  </form>
  <form class="theia-resend" action="${url.loginAction}" method="post" accept-charset="UTF-8">
    <input type="hidden" name="smsAction" value="resend" />
    <span class="theia-resend__caption">کد رو دریافت نکردی؟</span>
    <button type="submit" id="theia-resend-button" data-wait-seconds="${(smsResendRemaining!0)?c}">ارسال دوباره کد</button>
    <span id="theia-resend-timer" class="theia-resend__time" aria-live="off"></span>
    <span id="theia-resend-ready" class="theia-visually-hidden" role="status" aria-live="polite"></span>
  </form>
</@layout.page>
