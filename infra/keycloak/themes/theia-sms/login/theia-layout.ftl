<#ftl output_format="HTML" auto_esc=true>
<#-- Auth UI only: no upstream Keycloak layout, inline scripts, remote assets, or login tokens. -->
<#macro page title step>
<!doctype html>
<html lang="fa" dir="rtl">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
  <meta name="color-scheme" content="light" />
  <meta name="referrer" content="no-referrer" />
  <meta name="theme-color" content="#f3f2ee" />
  <title>${title} | تیا</title>
  <link rel="stylesheet" href="${url.resourcesPath}/css/theia-login.css" />
  <script defer src="${url.resourcesPath}/js/theia-login.js"></script>
</head>
<body class="theia-auth theia-auth--${step}">
  <main class="theia-frame">
    <div class="theia-surface">
      <header class="theia-brand" aria-label="تیا">
        <img class="theia-brand__mark" src="${url.resourcesPath}/img/theia-mark.jpg" width="50" height="50" alt="" />
        <span class="theia-brand__name">تیا</span>
      </header>
      <div class="theia-content">
        <#nested>
      </div>
      <footer class="theia-footer">
        <span class="theia-footer__line" aria-hidden="true"></span>
        <p>یه همراه برای وقت‌هایی که می‌خوای حرف بزنی.</p>
      </footer>
    </div>
  </main>
</body>
</html>
</#macro>
