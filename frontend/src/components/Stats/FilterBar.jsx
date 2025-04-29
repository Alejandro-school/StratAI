import React, { useState } from 'react';
import '../../styles/Stats/FilterBar.css'; // Importa el CSS aquí

const FilterBar = ({ onFilterChange, availableMaps }) => {
  const [selectedMap, setSelectedMap] = useState('');
  const [dateRange, setDateRange] = useState({ start: '', end: '' });
  const [durationRange, setDurationRange] = useState({ min: 0, max: 60 });

  const handleMapChange = (e) => {
    const newMap = e.target.value;
    setSelectedMap(newMap);
    onFilterChange({ map: newMap, dateRange, durationRange });
  };

  const handleStartDateChange = (e) => {
    const newDateRange = { ...dateRange, start: e.target.value };
    setDateRange(newDateRange);
    onFilterChange({ map: selectedMap, dateRange: newDateRange, durationRange });
  };

  const handleEndDateChange = (e) => {
    const newDateRange = { ...dateRange, end: e.target.value };
    setDateRange(newDateRange);
    onFilterChange({ map: selectedMap, dateRange: newDateRange, durationRange });
  };

  const handleMinDurationChange = (e) => {
    const newDurationRange = { ...durationRange, min: Number(e.target.value) };
    setDurationRange(newDurationRange);
    onFilterChange({ map: selectedMap, dateRange, durationRange: newDurationRange });
  };

  const handleMaxDurationChange = (e) => {
    const newDurationRange = { ...durationRange, max: Number(e.target.value) };
    setDurationRange(newDurationRange);
    onFilterChange({ map: selectedMap, dateRange, durationRange: newDurationRange });
  };

  return (
    <div className="filter-bar">
      {/* Grupo para seleccionar el mapa */}
      <div className="filter-group">
        <label className="filter-label">Mapa</label>
        <select
          className="filter-select"
          value={selectedMap}
          onChange={handleMapChange}
        >
          <option value="">Todos</option>
          {availableMaps.map((map) => (
            <option key={map} value={map}>{map}</option>
          ))}
        </select>
      </div>

      {/* Fecha inicio */}
      <div className="filter-group">
        <label className="filter-label">Desde</label>
        <input
          type="date"
          className="filter-input"
          value={dateRange.start}
          onChange={handleStartDateChange}
        />
      </div>

      {/* Fecha fin */}
      <div className="filter-group">
        <label className="filter-label">Hasta</label>
        <input
          type="date"
          className="filter-input"
          value={dateRange.end}
          onChange={handleEndDateChange}
        />
      </div>

      {/* Rango de duración */}
      <div className="filter-group">
        <label className="filter-label">Duración (min)</label>
        <div style={{ display: 'flex', gap: '0.5rem' }}>
          <input
            type="number"
            className="filter-input"
            value={durationRange.min}
            onChange={handleMinDurationChange}
            style={{ width: '60px' }}
          />
          <input
            type="number"
            className="filter-input"
            value={durationRange.max}
            onChange={handleMaxDurationChange}
            style={{ width: '60px' }}
          />
        </div>
      </div>
    </div>
  );
};

export default FilterBar;
