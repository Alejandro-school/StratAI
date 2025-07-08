import React from 'react';
import useFadeInOnScroll from '../../hooks/useFadeInOnScroll';

const Comparison = () => {
  const ref = useFadeInOnScroll();
  return (
    <section ref={ref} className="fade-section">
      <div className="container">
        <h2>Antes y Despu\u00e9s</h2>
        <table className="comparison-table">
          <thead>
            <tr>
              <th></th>
              <th>Sin StratAI</th>
              <th>Con StratAI</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>K/D</td>
              <td>0.85</td>
              <td>1.24</td>
            </tr>
            <tr>
              <td>Errores</td>
              <td>23</td>
              <td>7</td>
            </tr>
            <tr>
              <td>Repeticiones</td>
              <td>Sin repeticiones</td>
              <td>Visi\u00f3n 2D ronda a ronda</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  );
};

export default Comparison;
