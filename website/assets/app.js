/* ============================================================
   GANTRY LANDING PAGE — APP.JS
   ============================================================ */

/* — Nav (no scroll class needed; pill is always styled) — */

/* — Sliding nav section indicator — */
(function () {
  const linksContainer = document.querySelector('.nav-links');
  if (!linksContainer) return;

  // Inject sliding pill
  const pill = document.createElement('div');
  pill.className = 'nav-indicator';
  linksContainer.appendChild(pill);

  // Map anchor nav links → their target sections
  const navItems = Array.from(document.querySelectorAll('.nav-link[href^="#"]')).map(link => ({
    link,
    section: document.getElementById(link.getAttribute('href').slice(1))
  })).filter(item => item.section);

  let currentLink = null;

  function moveToLink(link) {
    if (link === currentLink) return;
    currentLink = link;
    if (!link) {
      pill.style.opacity = '0';
      return;
    }
    const cRect = linksContainer.getBoundingClientRect();
    const lRect = link.getBoundingClientRect();
    pill.style.left   = (lRect.left - cRect.left) + 'px';
    pill.style.width  = lRect.width + 'px';
    pill.style.top    = (lRect.top  - cRect.top)  + 'px';
    pill.style.height = lRect.height + 'px';
    pill.style.opacity = '1';
  }

  function onScroll() {
    const scrollMid = window.scrollY + 140; // offset below fixed nav
    let active = null;
    for (const { link, section } of navItems) {
      const top = section.getBoundingClientRect().top + window.scrollY;
      if (scrollMid >= top && scrollMid < top + section.offsetHeight) {
        active = link;
      }
    }
    moveToLink(active);
  }

  window.addEventListener('scroll', onScroll, { passive: true });
  onScroll();
})();

/* — Scroll reveal — */
const revealObserver = new IntersectionObserver(entries => {
  entries.forEach(e => { if (e.isIntersecting) e.target.classList.add('visible'); });
}, { threshold: 0.08, rootMargin: '0px 0px -40px 0px' });

document.querySelectorAll('.reveal').forEach(el => revealObserver.observe(el));

/* — Terminal typewriter — */
const termEl = document.getElementById('terminal-content');

const LINES = [
  { cls: 't-prompt', text: '$ gantry serve --dev', pause: 400 },
  { cls: 't-dim',    text: '', pause: 180 },
  { cls: 't-dim',    text: '  → Gantry v0.1.0', pause: 100 },
  { cls: 't-ok',     text: '  ✓ Database initialized', pause: 110 },
  { cls: 't-ok',     text: '  ✓ Schema validation ready', pause: 100 },
  { cls: 't-ok',     text: '  ✓ Kubernetes plugin  (3 clusters)', pause: 150 },
  { cls: 't-ok',     text: '  ✓ GitHub plugin connected', pause: 110 },
  { cls: 't-ok',     text: '  ✓ ArgoCD plugin connected', pause: 100 },
  { cls: 't-dim',    text: '', pause: 90 },
  { cls: 't-ok',     text: '  ✓ 127 entities loaded', pause: 180 },
  { cls: 't-dim',    text: '', pause: 80 },
  { cls: 't-url',    text: '  → Listening on http://localhost:8080', pause: 0 },
];

async function wait(ms) { return new Promise(r => setTimeout(r, ms)); }

async function runTerminal() {
  await wait(600);
  for (const line of LINES) {
    await wait(line.pause);
    const span = document.createElement('span');
    span.className = `t-line ${line.cls}`;
    span.textContent = line.text;
    termEl.appendChild(span);
  }
  // blinking cursor
  const cur = document.createElement('span');
  cur.className = 't-cursor';
  termEl.appendChild(cur);
}

// Only animate when terminal enters viewport
const termObserver = new IntersectionObserver(entries => {
  if (entries[0].isIntersecting) {
    runTerminal();
    termObserver.disconnect();
  }
}, { threshold: 0.25 });

if (termEl) termObserver.observe(termEl);

/* — Smooth anchor scroll — */
document.querySelectorAll('a[href^="#"]').forEach(a => {
  a.addEventListener('click', e => {
    const target = document.querySelector(a.getAttribute('href'));
    if (target) { e.preventDefault(); target.scrollIntoView({ behavior: 'smooth', block: 'start' }); }
  });
});
