import React, { useEffect, useState } from 'react';
import axios from 'axios';

const HistoryGames = () => {
  const [shareCodes, setShareCodes] = useState([]);
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState([]);

  useEffect(() => {
    const fetchShareCodes = async () => {
      try {
        const response = await axios.get('http://localhost:8000/steam/get-saved-sharecodes', {
          withCredentials: true,
        });
        setShareCodes(response.data.sharecodes);
      } catch (err) {
        console.error("❌ Error al cargar los share codes:", err);
      }
    };

    fetchShareCodes();
  }, []);

  const handleProcessDemos = async () => {
    setLoading(true);
    try {
      const response = await axios.post('http://localhost:8000/steam/process-demos', {
        sharecodes: shareCodes,
      }, { withCredentials: true });

      setStats(response.data.processed_stats);
    } catch (err) {
      console.error("❌ Error al procesar demos:", err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <h2>📂 Partidas Guardadas</h2>
      <button onClick={handleProcessDemos} disabled={loading}>
        {loading ? 'Procesando...' : 'Procesar Demos'}
      </button>
      <ul>
        {stats.map((stat, index) => (
          <li key={index}>
            <strong>{stat.share_code}</strong>
            <p>Kills: {stat.stats.kills}</p>
            <p>Deaths: {stat.stats.deaths}</p>
            <p>Assists: {stat.stats.assists}</p>
            <p>Headshots: {stat.stats.headshots}</p>
            <p>Rounds: {stat.stats.rounds_played}</p>
          </li>
        ))}
      </ul>
    </div>
  );
};

export default HistoryGames;
