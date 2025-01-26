import React, { useState } from 'react';
import axios from 'axios';
import StatsChart from './StatsChart';

import '../styles/common.css';


const UploadDemo = () => {
  const [selectedFile, setSelectedFile] = useState(null);
  const [matchId, setMatchId] = useState('Match1');
  const [statsData, setStatsData] = useState(null);
  const [matches, setMatches] = useState([]);

  const handleFileChange = (e) => {
    setSelectedFile(e.target.files[0]);
  };

  const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

  const handleUpload = async () => {
    if (!selectedFile || !selectedFile.name.endsWith('.dem')) {
      alert('Por favor, selecciona un archivo .dem válido');
      return;
    }

    const uniqueId = `Match_${Date.now()}`;

    const formData = new FormData();
    formData.append('file', selectedFile);

    try {
      const response = await axios.post(
        `${API_URL}/process-demo`,
        formData,
        { headers: { 'Content-Type': 'multipart/form-data' } }
      );

      const newMatch = {
        id: uniqueId,
        name: uniqueId,
        stats: response.data
      };

      setMatches((prevMatches) => [...prevMatches, newMatch]);
      setStatsData(response.data);
      setMatchId(uniqueId);
    } catch (error) {
      console.error('Error al procesar demo:', error);
      alert('Error al procesar el archivo demo.');
    }
  };

  const handleMatchSelect = (id) => {
    const selectedMatch = matches.find((match) => match.id === id);
    if (selectedMatch) {
      setStatsData(selectedMatch.stats);
    }
  };

  return (
    <div className="upload-demo-container">
      <input type="file" accept=".dem" onChange={handleFileChange} />
      <br />
      <button onClick={handleUpload}>Procesar Demo</button>

      <div className="match-select">
        <select
          value={matchId}
          onChange={(e) => handleMatchSelect(e.target.value)}
        >
          {matches.map((match) => (
            <option key={match.id} value={match.id}>
              {match.name}
            </option>
          ))}
        </select>
      </div>

      {statsData && (
        <StatsChart
          stats={statsData}
          matches={matches}
          onMatchSelect={handleMatchSelect}
        />
      )}
    </div>
  );
};

export default UploadDemo;
