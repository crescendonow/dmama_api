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

  // ============ Customers-all Playground ============
  // See note/22_plan_for_customers_endpoint.md ("Playground" row): only this endpoint gets an
  // interactive playground, because it streams. The API key lives only in the ca-api-key input's
  // in-memory value -- never written to the URL, a query string, or localStorage -- and is lost on
  // refresh. Requests use a URL relative to this docs page (../api/dma/customers-all) so it works
  // both at /docs/ and behind nginx at /dmama_api/docs/.
  (function () {
    var form = document.getElementById('customers-all-form');
    if (!form) return; // only present on index.html

    var apiKeyInput = document.getElementById('ca-api-key');
    var regionInput = document.getElementById('ca-region');
    var pwaCodeInput = document.getElementById('ca-pwa-code');
    var dmaIdInput = document.getElementById('ca-dma-id');
    var usetypeInput = document.getElementById('ca-usetype');
    var polygonInput = document.getElementById('ca-polygon');
    var statusEl = document.getElementById('ca-status');
    var outputEl = document.getElementById('ca-output');
    var downloadBtn = document.getElementById('ca-download');

    var PREVIEW_LIMIT_BYTES = 200 * 1024; // ~200 KB, per plan
    var CUSTOMERS_ALL_URL = '../api/dma/customers-all';
    var lastRequest = null; // set on submit; reused by the Download button so it matches the preview

    // buildRequest turns the form fields into a plain { method, url, headers, body } request
    // description. A non-empty my_polygon means POST with every filter in the JSON body; an empty
    // one means GET with the filters as query params (see the plan's parameter hierarchy).
    function buildRequest() {
      var params = new URLSearchParams();
      var addParam = function (name, value) {
        var trimmed = (value || '').trim();
        if (trimmed) params.append(name, trimmed);
      };
      addParam('region', regionInput.value);
      addParam('pwa_code', pwaCodeInput.value);
      addParam('dma_id', dmaIdInput.value);
      addParam('usetype', usetypeInput.value);

      var polygonText = (polygonInput.value || '').trim();
      if (!polygonText) {
        var query = params.toString();
        return {
          method: 'GET',
          url: CUSTOMERS_ALL_URL + (query ? '?' + query : ''),
          headers: { 'X-API-Key': apiKeyInput.value }
        };
      }

      var body = {};
      if (regionInput.value.trim()) body.region = parseInt(regionInput.value, 10);
      if (pwaCodeInput.value.trim()) body.pwa_code = pwaCodeInput.value.trim();
      if (dmaIdInput.value.trim()) {
        body.dma_id = dmaIdInput.value.split(',')
          .map(function (part) { return parseInt(part.trim(), 10); })
          .filter(function (n) { return !isNaN(n); });
      }
      if (usetypeInput.value.trim()) {
        body.usetype = usetypeInput.value.split(',')
          .map(function (part) { return part.trim(); })
          .filter(Boolean);
      }
      try {
        body.my_polygon = JSON.parse(polygonText);
      } catch (parseError) {
        throw new Error('my_polygon ต้องเป็น GeoJSON ที่ valid: ' + parseError.message);
      }
      return {
        method: 'POST',
        url: CUSTOMERS_ALL_URL,
        headers: { 'X-API-Key': apiKeyInput.value, 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      };
    }

    function fetchRequest(req) {
      return fetch(req.url, { method: req.method, headers: req.headers, body: req.body, cache: 'no-store' });
    }

    // runPreview streams the response via response.body.getReader(), showing at most ~200 KB, then
    // aborts (AbortController) so the browser and server both stop early instead of buffering (or
    // sending) a possibly multi-GB body just for a preview.
    function runPreview(req) {
      var controller = new AbortController();
      return fetch(req.url, {
        method: req.method,
        headers: req.headers,
        body: req.body,
        cache: 'no-store',
        signal: controller.signal
      }).then(function (response) {
        statusEl.textContent = 'HTTP ' + response.status;
        if (!response.ok) {
          return response.text().then(function (text) {
            outputEl.textContent = text || ('(no body, HTTP ' + response.status + ')');
          });
        }
        if (!response.body || !response.body.getReader) {
          return response.text().then(function (text) {
            var truncated = text.length > PREVIEW_LIMIT_BYTES;
            outputEl.textContent = text.slice(0, PREVIEW_LIMIT_BYTES) +
              (truncated ? '\n\n… (พรีวิวถูกตัดที่ ~200 KB — ใช้ปุ่ม "Download full result" เพื่อดูทั้งหมด)' : '');
          });
        }

        var reader = response.body.getReader();
        var decoder = new TextDecoder();
        var received = 0;
        var text = '';

        function pump() {
          return reader.read().then(function (result) {
            if (result.done) {
              outputEl.textContent = text;
              return;
            }
            text += decoder.decode(result.value, { stream: true });
            received += result.value.length;
            if (received >= PREVIEW_LIMIT_BYTES) {
              outputEl.textContent = text + '\n\n… (พรีวิวถูกตัดที่ ~200 KB — ใช้ปุ่ม "Download full result" เพื่อดูทั้งหมด)';
              controller.abort();
              return reader.cancel().catch(function () { /* already aborted */ });
            }
            return pump();
          });
        }
        return pump();
      }).catch(function (err) {
        if (err && err.name === 'AbortError') return; // expected: preview limit reached
        statusEl.textContent = 'เรียก API ไม่สำเร็จ';
        outputEl.textContent = err instanceof Error ? err.message : String(err);
      });
    }

    form.addEventListener('submit', function (event) {
      event.preventDefault();
      statusEl.textContent = 'กำลังโหลด…';
      outputEl.textContent = '';
      var req;
      try {
        req = buildRequest();
      } catch (buildError) {
        statusEl.textContent = 'พารามิเตอร์ไม่ถูกต้อง';
        outputEl.textContent = buildError.message;
        return;
      }
      lastRequest = req;
      runPreview(req);
    });

    downloadBtn?.addEventListener('click', function () {
      var req;
      try {
        req = lastRequest || buildRequest();
      } catch (buildError) {
        statusEl.textContent = 'พารามิเตอร์ไม่ถูกต้อง';
        outputEl.textContent = buildError.message;
        return;
      }
      statusEl.textContent = 'กำลังดาวน์โหลด…';

      if (window.showSaveFilePicker) {
        // Pipe the response body straight to disk: memory stays flat even for the "all regions,
        // no filters" case (5,000,000+ rows).
        window.showSaveFilePicker({ suggestedName: 'customers-all.json' })
          .then(function (handle) {
            return handle.createWritable().then(function (writable) {
              return fetchRequest(req).then(function (response) {
                if (!response.ok || !response.body) {
                  return response.text().then(function (text) {
                    throw new Error(text || ('HTTP ' + response.status));
                  });
                }
                return response.body.pipeTo(writable);
              });
            });
          })
          .then(function () {
            statusEl.textContent = 'บันทึกไฟล์เรียบร้อย';
          })
          .catch(function (err) {
            if (err && err.name === 'AbortError') {
              statusEl.textContent = 'ยกเลิกการบันทึก';
              return;
            }
            statusEl.textContent = 'ดาวน์โหลดไม่สำเร็จ';
            outputEl.textContent = err instanceof Error ? err.message : String(err);
          });
        return;
      }

      // Fallback for browsers without showSaveFilePicker (e.g. Firefox/Safari): buffer the whole
      // response as a Blob, then trigger a normal download. Note in the UI: very large results
      // (no filters = all regions) need the Chromium path above or curl -N instead.
      fetchRequest(req)
        .then(function (response) {
          if (!response.ok) {
            return response.text().then(function (text) {
              throw new Error(text || ('HTTP ' + response.status));
            });
          }
          return response.blob();
        })
        .then(function (blob) {
          var url = URL.createObjectURL(blob);
          var link = document.createElement('a');
          link.href = url;
          link.download = 'customers-all.json';
          document.body.appendChild(link);
          link.click();
          link.remove();
          URL.revokeObjectURL(url);
          statusEl.textContent = 'บันทึกไฟล์เรียบร้อย';
        })
        .catch(function (err) {
          statusEl.textContent = 'ดาวน์โหลดไม่สำเร็จ (ไฟล์ใหญ่มากอาจต้องใช้เบราว์เซอร์ที่รองรับ showSaveFilePicker หรือ curl -N แทน)';
          outputEl.textContent = err instanceof Error ? err.message : String(err);
        });
    });
  })();
});
