import React, { useEffect, useState } from 'react';
import '../../styles/Landing/SidebarNav.css';

/* ───────────── Ítems del menú ───────────── */
const NAV_ITEMS = [
  { id: 'inicio',          label: '01 Inicio' },
  { id: 'funcionalidades', label: '02 Funcionalidades' },
  { id: 'estadisticas',    label: '03 Estadísticas' },
  { id: 'recomendaciones', label: '04 Recomendaciones' },
  { id: 'contacto',        label: '05 Contacto' }
];

const clamp = (v, min = 0, max = 1) => Math.max(min, Math.min(max, v));

export default function SidebarNav () {
  const [activeId, setActiveId] = useState(NAV_ITEMS[0].id);

  useEffect(() => {
    /* refs a secciones y barras */
    const sections = NAV_ITEMS.map(({ id }) => document.getElementById(id));
    const items    = NAV_ITEMS.map(({ id }) => ({
      li  : document.querySelector(`li[data-id="${id}"]`),
      bar : document.querySelector(`li[data-id="${id}"] .progress`),
      lbl : document.querySelector(`li[data-id="${id}"] .label`)
    }));

    /* ancho de cada barra = ancho del texto */
    const resizeBars = () => {
      items.forEach(({ lbl, bar }) => {
        if (!lbl || !bar) return;
        bar.style.left  = `${lbl.offsetLeft}px`;
        bar.style.width = `${lbl.offsetWidth}px`;
      });
    };

    const onScroll = () => {
      const top        = window.scrollY;
      const vh         = window.innerHeight;
      const desktop    = window.innerWidth > 700;
      const docBottom  = top + vh >= document.documentElement.scrollHeight - 2;

      sections.forEach((sec, idx) => {
        if (!sec) return;

        const { li, bar } = items[idx];
        if (!bar) return;

        const start   = sec.offsetTop;
        const end     = start + sec.offsetHeight;
        const inView  = top >= start && top < end;           // borde sup. dentro
        let progress  = clamp((top - start) / sec.offsetHeight, 0, 1);

        /* último bloque: al tocar fondo fuerza 100 % */
        if (idx === sections.length - 1 && docBottom) progress = 1;

        /* ─── actualiza visual ─── */
        if (inView || (idx === sections.length - 1 && docBottom)) {
          bar.style.transform = desktop
            ? `scaleX(${progress})`
            : `scaleY(${progress})`;
          bar.style.opacity   = progress > 0 ? 1 : 0;
          li.classList.add('active');
          setActiveId(sec.id);
        } else {
          bar.style.opacity   = 0;
          bar.style.transform = desktop ? 'scaleX(0)' : 'scaleY(0)';
          li.classList.remove('active');
        }
      });
    };

    /* listeners */
    resizeBars();
    onScroll();
    const handleResize = () => { resizeBars(); onScroll(); };
    window.addEventListener('resize', handleResize);
    window.addEventListener('scroll',  onScroll, { passive: true });

    return () => {
      window.removeEventListener('resize', handleResize);
      window.removeEventListener('scroll',  onScroll);
    };
  }, []);

  /* scroll suave al clicar */
  const goTo = id => {
    const sec = document.getElementById(id);
    if (sec) sec.scrollIntoView({ behavior: 'smooth', block: 'start' });
  };

  return (
    <aside className="sidebar-nav">
      <nav>
        <ul>
          {NAV_ITEMS.map(({ id, label }) => (
            <li key={id}
                data-id={id}
                className={activeId === id ? 'active' : ''}
                onClick={() => goTo(id)}
            >
              <span className="label">{label}</span>
              <span className="progress" />
            </li>
          ))}
        </ul>
      </nav>
    </aside>
  );
}
