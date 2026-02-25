// ── Theme Toggle ────────────────────────────────────────────────────

(function () {
  const html = document.documentElement;
  const toggle = document.getElementById('theme-toggle');

  function getPreferred() {
    const stored = localStorage.getItem('theme');
    if (stored) return stored;
    return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
  }

  function applyTheme(theme) {
    html.setAttribute('data-theme', theme);
    localStorage.setItem('theme', theme);
  }

  applyTheme(getPreferred());

  toggle.addEventListener('click', function () {
    var current = html.getAttribute('data-theme');
    var next = current === 'dark' ? 'light' : 'dark';
    applyTheme(next);

    // Rotate icon
    var icon = current === 'dark' ? toggle.querySelector('.icon-sun') : toggle.querySelector('.icon-moon');
    if (icon) {
      icon.style.transform = 'rotate(180deg)';
      setTimeout(function () { icon.style.transform = ''; }, 300);
    }
  });
})();

// ── Copy to Clipboard ───────────────────────────────────────────────

document.querySelectorAll('.copy-btn').forEach(function (btn) {
  btn.addEventListener('click', function () {
    var text = btn.getAttribute('data-copy');
    if (!text) return;

    navigator.clipboard.writeText(text).then(function () {
      btn.classList.add('copied');
      setTimeout(function () {
        btn.classList.remove('copied');
      }, 2000);
    });
  });
});

// ── Nav Scroll Effect ───────────────────────────────────────────────

(function () {
  var nav = document.getElementById('nav');
  var scrolled = false;

  window.addEventListener('scroll', function () {
    var shouldBeScrolled = window.scrollY > 50;
    if (shouldBeScrolled !== scrolled) {
      scrolled = shouldBeScrolled;
      nav.classList.toggle('scrolled', scrolled);
    }
  }, { passive: true });
})();

// ── Mobile Nav Toggle ───────────────────────────────────────────────

(function () {
  var toggle = document.getElementById('nav-mobile-toggle');
  var links = document.querySelector('.nav-links');

  if (toggle && links) {
    toggle.addEventListener('click', function () {
      links.classList.toggle('open');
    });

    // Close on link click
    links.querySelectorAll('a').forEach(function (a) {
      a.addEventListener('click', function () {
        links.classList.remove('open');
      });
    });
  }
})();

// ── Scroll Reveal ───────────────────────────────────────────────────

(function () {
  var elements = document.querySelectorAll('.reveal');

  var observer = new IntersectionObserver(function (entries) {
    entries.forEach(function (entry) {
      if (entry.isIntersecting) {
        // Stagger steps within timeline
        var el = entry.target;
        var step = el.closest('.step');
        if (step) {
          var steps = Array.from(step.parentElement.children);
          var index = steps.indexOf(step);
          el.style.transitionDelay = (index * 100) + 'ms';
        }

        el.classList.add('visible');
        observer.unobserve(el);
      }
    });
  }, {
    threshold: 0.1,
    rootMargin: '0px 0px -50px 0px'
  });

  elements.forEach(function (el) {
    observer.observe(el);
  });
})();

// ── Consent Manager ────────────────────────────────────────────────

(function () {
  var GTM_ID = 'GTM-WRLZND6J';
  var banner = document.getElementById('consent-banner');
  var acceptBtn = document.getElementById('consent-accept');
  var rejectBtn = document.getElementById('consent-reject');

  function loadGTM() {
    // Update consent state
    gtag('consent', 'update', {
      'analytics_storage': 'granted'
    });

    // Inject GTM script
    (function (w, d, s, l, i) {
      w[l] = w[l] || [];
      w[l].push({ 'gtm.start': new Date().getTime(), event: 'gtm.js' });
      var f = d.getElementsByTagName(s)[0],
        j = d.createElement(s),
        dl = l != 'dataLayer' ? '&l=' + l : '';
      j.async = true;
      j.src = 'https://www.googletagmanager.com/gtm.js?id=' + i + dl;
      f.parentNode.insertBefore(j, f);
    })(window, document, 'script', 'dataLayer', GTM_ID);
  }

  function hideBanner() {
    banner.classList.add('hidden');
  }

  function showBanner() {
    banner.style.display = '';
  }

  // Check stored consent
  var consent = localStorage.getItem('consent');

  if (consent === 'accepted') {
    banner.style.display = 'none';
    loadGTM();
  } else if (consent === 'rejected') {
    banner.style.display = 'none';
  } else {
    showBanner();
  }

  acceptBtn.addEventListener('click', function () {
    localStorage.setItem('consent', 'accepted');
    hideBanner();
    loadGTM();
  });

  rejectBtn.addEventListener('click', function () {
    localStorage.setItem('consent', 'rejected');
    hideBanner();
  });
})();
