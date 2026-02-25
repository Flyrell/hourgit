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
