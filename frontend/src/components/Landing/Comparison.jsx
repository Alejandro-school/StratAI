// Archivo: Comparison.jsx
import React from 'react';
import styles from '../../styles/Landing/Comparison.module.css';

const Comparison = () => {
  return (
    <section className={styles.section}>
      <div className={styles.container}>
        <h2 className={styles.heading}>¿Por qué StratAI?</h2>
        <div className={styles.tableWrapper}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Característica</th>
                <th>Trackers Tradicionales</th>
                <th>StratAI</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>Estadísticas Básicas</td>
                <td>✔️</td>
                <td>✔️</td>
              </tr>
              <tr>
                <td>Análisis de errores estratégicos</td>
                <td>❌</td>
                <td>✔️</td>
              </tr>
              <tr>
                <td>Coaching personalizado con IA</td>
                <td>❌</td>
                <td>✔️</td>
              </tr>
              <tr>
                <td>Visualización táctica</td>
                <td>❌</td>
                <td>✔️</td>
              </tr>
              <tr>
                <td>Recomendaciones automáticas</td>
                <td>❌</td>
                <td>✔️</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
};

export default Comparison;