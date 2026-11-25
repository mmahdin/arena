<#ftl output_format="HTML" auto_esc=true>
<#import "theia-layout.ftl" as layout>
<@layout.page title="ورود به حساب" step="phone">
  <section class="theia-intro" aria-labelledby="theia-title">
    <span class="theia-eyebrow">ورود به تیا</span>
    <h1 id="theia-title">خوش اومدی!</h1>
    <p>برای ورود یا ساخت حساب، شماره موبایلت رو وارد کن.</p>
  </section>

  <#if smsError?? && smsError?has_content>
    <div id="theia-auth-error" class="theia-error" role="alert">
      <span class="theia-error__icon" aria-hidden="true">!</span>
      <span>${smsError}</span>
    </div>
  </#if>

  <form id="theia-phone-form" class="theia-form" action="${url.loginAction}" method="post" accept-charset="UTF-8">
    <input type="hidden" name="smsAction" value="request" />
    <div class="theia-field">
      <label for="theia-phone">شماره موبایل</label>
      <div class="theia-phone-wrap">
        <input id="theia-phone" name="phone" type="tel" dir="ltr" inputmode="tel"
               autocomplete="tel-national" enterkeyhint="go" spellcheck="false"
               placeholder="0912 345 6789" required maxlength="20"
               <#if smsEnteredPhone?? && smsEnteredPhone?has_content>value="${smsEnteredPhone}"</#if>
               aria-describedby="theia-phone-help<#if smsError?? && smsError?has_content> theia-auth-error</#if>"
               <#if smsError?? && smsError?has_content>aria-invalid="true"</#if> />
        <span class="theia-field__glyph" aria-hidden="true">
          <svg width="19" height="20" viewBox="0 0 20 20" fill="none"><rect x="5.5" y="1.9" width="9" height="16.2" rx="2" stroke="currentColor" stroke-width="1.5"/><path d="M8.5 14.9h3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
        </span>
      </div>
      <p id="theia-phone-help" class="theia-field__hint">کد ورود به این شماره پیامک می‌شه.</p>
    </div>
    <button class="theia-primary" type="submit">دریافت کد تأیید <span class="theia-primary__arrow" aria-hidden="true">←</span></button>
  </form>
  <p class="theia-under-note">شماره موبایلت، شناسه ورود به حساب تیاست.</p>
</@layout.page>
