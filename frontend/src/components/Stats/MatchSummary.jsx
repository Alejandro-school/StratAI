import React, { useEffect, useState } from 'react';
import { Bar } from 'react-chartjs-2';
import axios from 'axios';
import '../../styles/Stats/matchSummary.css';

const API_URL = process.env.REACT_APP_API_URL || (window.location.port === '3000' ? 'http://localhost:8000' : '');

// Registro de los componentes necesarios de Chart.js
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend
} from 'chart.js';

// Registrar componentes
ChartJS.register(CategoryScale, LinearScale, BarElement, Title, Tooltip, Legend);

const MatchSummary = () => {
  const [stats, setStats] = useState([]);

  useEffect(() => {
    axios.get(`${API_URL}/steam/match-history`, { withCredentials: true })
    .then(response => setStats(response.data.stats))
    .catch(error => console.error('Error al cargar las estadísticas:', error));  
  }, []);

  // Filtrar estadísticas relevantes (Kills y Deaths)
  const totalKills = stats.find(stat => stat.name === 'total_kills')?.value || 0;
  const totalDeaths = stats.find(stat => stat.name === 'total_deaths')?.value || 0;
  const totalTime = stats.find(stat => stat.name === 'total_time_played')?.value || 0;


  const chartData = {
    labels: ['Kills', 'Deaths'],
    datasets: [
      {
        label: 'Estadísticas Generales',
        data: [totalKills, totalDeaths],
        backgroundColor: ['rgba(75, 192, 192, 0.6)', 'rgba(255, 99, 132, 0.6)'],
        borderColor: ['rgba(75, 192, 192, 1)', 'rgba(255, 99, 132, 1)'],
        borderWidth: 1,
      },
    ],
  };

  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    scales: {
      y: {
        beginAtZero: true,
      },
    },
    plugins: {
      legend: { position: 'top' },
      title: { display: true, text: 'Kills vs Deaths' },
    },
  };

  return (
    <div className="match-summary-container">
      <h2>Resumen de Partidas</h2>

      <div className="match-cards">
        <div className="match-card">
          <h3>Kills totales</h3>
          <p>{totalKills}</p>
        </div>
        <div className="match-card">
          <h3>Muertes totales</h3>
          <p>{totalDeaths}</p>
        </div>
      </div>

      <div className="match-chart">
        <h3>Rendimiento General</h3>
        <div style={{ height: '400px' }}>
          <Bar data={chartData} options={chartOptions} redraw />
        </div>
      </div>
    </div>
  );
};

export default MatchSummary;
