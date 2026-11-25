<#ftl output_format="HTML" auto_esc=true>
<#import "theia-layout.ftl" as layout>
<@layout.page title="خروج از حساب" step="logout">
  <section class="theia-intro" aria-labelledby="theia-title">
    <h1 id="theia-title">خروج از حساب</h1>
    <p>برای تکمیل خروج از تیا در این مرورگر، خروج رو تأیید کن.</p>
  </section>

  <form class="theia-form" action="${url.logoutConfirmAction}" method="post">
    <input type="hidden" name="session_code" value="${logoutConfirm.code}" />
    <button class="theia-primary" type="submit" name="confirmLogout" id="kc-logout" value="true">تأیید خروج</button>
  </form>
  <#if !logoutConfirm.skipLink && (client.baseUrl)?has_content>
    <p class="theia-under-note"><a href="${client.baseUrl}">بازگشت به تیا</a></p>
  </#if>
</@layout.page>
