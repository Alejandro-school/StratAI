import React, { useRef } from 'react';
import { useGsapFadeIn } from '../../hooks/useGsapFadeIn';
import styles from '../../styles/Landing/Team.module.css';

const team = [
  { name: 'Lucía R.', role: 'Founder & IGL', img: '/team/lucia.jpg' },
  { name: 'Carlos M.', role: 'Data Scientist', img: '/team/carlos.jpg' },
  { name: 'Sara T.', role: 'Frontend Lead', img: '/team/sara.jpg' },
];

const Team = () => {
  const ref = useRef(null);
  useGsapFadeIn(ref, { y: 50 });

  return (
    <section id="team" ref={ref} className={styles.section}>
      <h2 className={styles.title}>Nuestro equipo</h2>
      <div className={styles.grid}>
        {team.map(({ name, role, img }) => (
          <div key={name} className={styles.card}>
            <img src={img} alt={name} />
            <h3>{name}</h3>
            <p>{role}</p>
          </div>
        ))}
      </div>
    </section>
  );
};

export default Team;
