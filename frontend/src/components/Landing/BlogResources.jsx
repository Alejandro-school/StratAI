import React, { useRef } from 'react';
import { useGsapFadeIn } from '../../hooks/useGsapFadeIn';
import styles from '../../styles/Landing/BlogResources.module.css';

const posts = [
  { title: 'Cómo dominar Inferno en CT-side', date: '13 Mar 2024', img: '/blog/inferno.jpg' },
  { title: 'Economía: cuándo comprar AWP', date: '28 Feb 2024', img: '/blog/awp.jpg' },
  { title: 'Top 5 granadas meta 2024', date: '10 Feb 2024', img: '/blog/nades.jpg' },
];

const BlogResources = () => {
  const ref = useRef(null);
  useGsapFadeIn(ref, { y: 40 });

  return (
    <section id="blog" ref={ref} className={styles.section}>
      <h2 className={styles.title}>Últimos artículos</h2>
      <div className={styles.grid}>
        {posts.map(({ title, date, img }) => (
          <article key={title} className={styles.card}>
            <img src={img} alt={title} />
            <div className={styles.content}>
              <span className={styles.date}>{date}</span>
              <h3>{title}</h3>
            </div>
          </article>
        ))}
      </div>
    </section>
  );
};

export default BlogResources;
