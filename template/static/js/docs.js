document.addEventListener('DOMContentLoaded', function () {
  // ============ Dark Mode ============
  const themeToggle = document.getElementById('theme-toggle');
  const iconSun = document.getElementById('icon-sun');
  const iconMoon = document.getElementById('icon-moon');

  function setTheme(dark) {
    document.documentElement.classList.toggle('dark', dark);
    iconSun.classList.toggle('hidden', !dark);
    iconMoon.classList.toggle('hidden', dark);
    localStorage.setItem('theme', dark ? 'dark' : 'light');
  }

  // Init from localStorage or system preference
  const saved = localStorage.getItem('theme');
  if (saved === 'dark' || (!saved && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    setTheme(true);
  }

  themeToggle.addEventListener('click', function () {
    setTheme(!document.documentElement.classList.contains('dark'));
  });

  // ============ Mobile Sidebar ============
  const sidebar = document.getElementById('sidebar');
  const sidebarToggle = document.getElementById('sidebar-toggle');
  const sidebarOverlay = document.getElementById('sidebar-overlay');

  function openSidebar() {
    sidebar.classList.remove('-translate-x-full');
    sidebarOverlay.classList.remove('hidden');
  }

  function closeSidebar() {
    sidebar.classList.add('-translate-x-full');
    sidebarOverlay.classList.add('hidden');
  }

  sidebarToggle.addEventListener('click', function () {
    if (sidebar.classList.contains('-translate-x-full')) {
      openSidebar();
    } else {
      closeSidebar();
    }
  });

  sidebarOverlay.addEventListener('click', closeSidebar);

  // Close sidebar on nav click (mobile)
  sidebar.querySelectorAll('.nav-link').forEach(function (link) {
    link.addEventListener('click', function () {
      if (window.innerWidth < 1024) {
        closeSidebar();
      }
    });
  });

  // ============ Collapsible Nav Groups ============
  document.querySelectorAll('.nav-group-toggle').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var items = btn.nextElementSibling;
      var arrow = btn.querySelector('.nav-arrow');
      items.classList.toggle('hidden');
      arrow.classList.toggle('rotated');
    });
  });

  // ============ Active Section Tracking ============
  var navLinks = document.querySelectorAll('.nav-link');
  var sections = [];

  navLinks.forEach(function (link) {
    var href = link.getAttribute('href');
    if (href && href.startsWith('#')) {
      var section = document.getElementById(href.slice(1));
      if (section) {
        sections.push({ el: section, link: link, id: href.slice(1) });
      }
    }
  });

  function setActiveLink(link) {
    navLinks.forEach(function (l) { l.classList.remove('active'); });
    if (link) {
      link.classList.add('active');
      // Auto-expand parent group if collapsed
      var group = link.closest('.nav-group-items');
      if (group && group.classList.contains('hidden')) {
        group.classList.remove('hidden');
        var arrow = group.previousElementSibling.querySelector('.nav-arrow');
        if (arrow) arrow.classList.add('rotated');
      }
    }
  }

  // IntersectionObserver for scroll tracking
  if ('IntersectionObserver' in window) {
    var visibleSections = new Map();

    var observer = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (entry.isIntersecting) {
          visibleSections.set(entry.target.id, entry.intersectionRatio);
        } else {
          visibleSections.delete(entry.target.id);
        }
      });

      // Find the topmost visible section
      var topSection = null;
      var topY = Infinity;
      visibleSections.forEach(function (_ratio, id) {
        var el = document.getElementById(id);
        if (el) {
          var rect = el.getBoundingClientRect();
          if (rect.top < topY && rect.top >= -100) {
            topY = rect.top;
            topSection = id;
          }
        }
      });

      if (topSection) {
        var match = sections.find(function (s) { return s.id === topSection; });
        if (match) setActiveLink(match.link);
      }
    }, {
      rootMargin: '-80px 0px -60% 0px',
      threshold: [0, 0.1, 0.5]
    });

    sections.forEach(function (s) { observer.observe(s.el); });
  }

  // ============ Copy Buttons ============
  document.querySelectorAll('.copy-btn').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var text = btn.getAttribute('data-copy');
      if (!text) {
        // Get text from sibling pre/code
        var codeBlock = btn.closest('.code-block');
        if (codeBlock) {
          var code = codeBlock.querySelector('code');
          if (code) text = code.textContent;
        }
      }

      if (text && navigator.clipboard) {
        navigator.clipboard.writeText(text).then(function () {
          var orig = btn.textContent;
          btn.textContent = 'Copied!';
          btn.classList.add('copied');
          setTimeout(function () {
            btn.textContent = orig;
            btn.classList.remove('copied');
          }, 2000);
        });
      }
    });
  });

  // ============ Smooth Scroll ============
  document.querySelectorAll('a[href^="#"]').forEach(function (a) {
    a.addEventListener('click', function (e) {
      var target = document.getElementById(a.getAttribute('href').slice(1));
      if (target) {
        e.preventDefault();
        target.scrollIntoView({ behavior: 'smooth', block: 'start' });
        history.pushState(null, '', a.getAttribute('href'));
      }
    });
  });

  // ============ Init: expand groups with active items on page load ============
  if (window.location.hash) {
    var hash = window.location.hash.slice(1);
    var match = sections.find(function (s) { return s.id === hash; });
    if (match) {
      setActiveLink(match.link);
      setTimeout(function () {
        document.getElementById(hash).scrollIntoView({ block: 'start' });
      }, 100);
    }
  }
});
