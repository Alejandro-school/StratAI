import React, { useState, useEffect } from 'react';
import {
  BarChart, Bar, LineChart, Line, XAxis, YAxis, CartesianGrid,
  Tooltip, Legend, ResponsiveContainer
} from 'recharts';

const StatsChart = ({ stats, matches, onMatchSelect }) => {
  console.log('Datos recibidos:', stats); // Para depuración

  const [selectedPlayer, setSelectedPlayer] = useState('');
  const [selectedMatch, setSelectedMatch] = useState(matches.length > 0 ? matches[0].id : '');

  // Manejar selección de partida
  useEffect(() => {
    if (matches.length > 0 && !selectedMatch) {
      setSelectedMatch(matches[0].id);
      onMatchSelect(matches[0].id);
    }
  }, [matches, selectedMatch, onMatchSelect]);

  const handleMatchChange = (event) => {
    const matchId = event.target.value;
    setSelectedMatch(matchId);
    onMatchSelect(matchId);
    setSelectedPlayer(''); // Reiniciar selección de jugador
  };

  // Manejar selección de jugador
  const handlePlayerChange = (event) => {
    setSelectedPlayer(event.target.value);
  };

  // Preparar datos por jugador
  const playerOptions = stats.players ? Object.keys(stats.players) : [];
  const playerStats = selectedPlayer && stats.players[selectedPlayer]
    ? stats.players[selectedPlayer]
    : null;

  const playerData = playerStats
    ? [
        { name: 'Kills', value: playerStats.kills || 0 },
        { name: 'Assists', value: playerStats.assists || 0 },
        { name: 'Deaths', value: playerStats.deaths || 0 },
        { name: 'Headshots', value: playerStats.headshots || 0 },
        { name: 'Damage', value: playerStats.damage || 0 },
      ]
    : [];

  // Datos generales para gráficos
  const barData = [
    { name: 'Kills', value: stats.kills || 0 },
    { name: 'Assists', value: stats.assists || 0 },
    { name: 'Deaths', value: stats.deaths || 0 },
    { name: 'Headshots', value: stats.headshots || 0 },
    { name: 'Rounds', value: stats.rounds_played || 0 },
  ];

  const lineData = [
    { name: 'ADR', value: stats.adr || 0 },
    { name: 'HS%', value: stats.hs_percent || 0 },
    { name: 'K/D', value: stats.kd || 0 },
    { name: 'Impact', value: stats.impact_score || 0 },
  ];

  return (
    <div>
      <h2>Resumen de Estadísticas</h2>

      {/* Selector de partidas */}
      <div>
        <label>Seleccionar Partida: </label>
        <select value={selectedMatch} onChange={handleMatchChange}>
          {matches.map((match) => (
            <option key={match.id} value={match.id}>{match.name}</option>
          ))}
        </select>
      </div>

      <ResponsiveContainer width="100%" height={300}>
        <BarChart data={barData}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="name" />
          <YAxis />
          <Tooltip />
          <Legend />
          <Bar dataKey="value" fill="#8884d8" />
        </BarChart>
      </ResponsiveContainer>

      <h2>Métricas Avanzadas</h2>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={lineData}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="name" />
          <YAxis />
          <Tooltip />
          <Legend />
          <Line type="monotone" dataKey="value" stroke="#82ca9d" />
        </LineChart>
      </ResponsiveContainer>

      <h2>Estadísticas por Jugador</h2>
      <div>
        <label>Seleccionar Jugador: </label>
        <select value={selectedPlayer} onChange={handlePlayerChange}>
          <option value="">Seleccionar...</option>
          {playerOptions.map((player) => (
            <option key={player} value={player}>{player}</option>
          ))}
        </select>
      </div>

      {playerStats && (
        <ResponsiveContainer width="100%" height={300}>
          <BarChart data={playerData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="name" />
            <YAxis />
            <Tooltip />
            <Legend />
            <Bar dataKey="value" fill="#82ca9d" />
          </BarChart>
        </ResponsiveContainer>
      )}
    </div>
  );
};

export default StatsChart;
