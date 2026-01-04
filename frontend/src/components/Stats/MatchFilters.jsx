// frontend/src/components/Stats/MatchFilters.jsx
// Advanced filter bar with map icons and filter pills
import React, { useState } from 'react';
import { 
  Search, Filter, X, Calendar, Map, 
  ChevronDown, Check, LayoutGrid, List
} from 'lucide-react';

const MAP_CONFIGS = [
  { id: 'de_dust2', name: 'Dust II', color: '#c4a35a' },
  { id: 'de_mirage', name: 'Mirage', color: '#6b8e23' },
  { id: 'de_inferno', name: 'Inferno', color: '#cd5c5c' },
  { id: 'de_nuke', name: 'Nuke', color: '#4682b4' },
  { id: 'de_overpass', name: 'Overpass', color: '#8fbc8f' },
  { id: 'de_train', name: 'Train', color: '#deb887' },
  { id: 'de_vertigo', name: 'Vertigo', color: '#87ceeb' },
  { id: 'de_anubis', name: 'Anubis', color: '#d4af37' },
  { id: 'de_ancient', name: 'Ancient', color: '#228b22' },
];

const MatchFilters = ({ 
  onFilterChange, 
  onViewChange, 
  currentView,
  availableMaps = [],
  filters,
  setFilters
}) => {
  const [showMapDropdown, setShowMapDropdown] = useState(false);
  const [showDateDropdown, setShowDateDropdown] = useState(false);
  const [showResultDropdown, setShowResultDropdown] = useState(false);

  // Get active pills
  const activePills = [];
  if (filters.map) {
    const mapConfig = MAP_CONFIGS.find(m => m.id === filters.map);
    activePills.push({
      id: 'map',
      label: mapConfig?.name || filters.map.replace('de_', ''),
      color: mapConfig?.color
    });
  }
  if (filters.dateRange) {
    activePills.push({
      id: 'date',
      label: filters.dateRange
    });
  }
  if (filters.result) {
    activePills.push({
      id: 'result',
      label: filters.result === 'victory' ? 'Victoria' : 'Derrota',
      color: filters.result === 'victory' ? '#22c55e' : '#ef4444'
    });
  }

  const handleMapSelect = (mapId) => {
    const newFilters = { ...filters, map: filters.map === mapId ? null : mapId };
    setFilters(newFilters);
    onFilterChange(newFilters);
    setShowMapDropdown(false);
  };

  const handleResultSelect = (result) => {
    const newFilters = { ...filters, result: filters.result === result ? null : result };
    setFilters(newFilters);
    onFilterChange(newFilters);
    setShowResultDropdown(false);
  };

  const handleDateRangeSelect = (range) => {
    const newFilters = { ...filters, dateRange: filters.dateRange === range ? null : range };
    setFilters(newFilters);
    onFilterChange(newFilters);
    setShowDateDropdown(false);
  };

  const removePill = (pillId) => {
    const newFilters = { ...filters, [pillId]: null };
    setFilters(newFilters);
    onFilterChange(newFilters);
  };

  const clearAllFilters = () => {
    const newFilters = { map: null, dateRange: null, result: null, search: '' };
    setFilters(newFilters);
    onFilterChange(newFilters);
  };

  return (
    <div className="match-filters">
      <div className="filters-row">
        {/* Search Input */}
        <div className="search-input-wrapper">
          <Search size={16} className="search-icon" />
          <input
            type="text"
            placeholder="Buscar por mapa..."
            value={filters.search || ''}
            onChange={(e) => {
              const newFilters = { ...filters, search: e.target.value };
              setFilters(newFilters);
              onFilterChange(newFilters);
            }}
            className="search-input"
          />
        </div>

        {/* Filter Dropdowns */}
        <div className="filter-buttons">
          {/* Map Filter */}
          <div className="filter-dropdown-wrapper">
            <button 
              className={`filter-btn ${filters.map ? 'active' : ''}`}
              onClick={() => {
                setShowMapDropdown(!showMapDropdown);
                setShowDateDropdown(false);
                setShowResultDropdown(false);
              }}
            >
              <Map size={16} />
              <span>Mapa</span>
              <ChevronDown size={14} className={showMapDropdown ? 'rotated' : ''} />
            </button>
            
            {showMapDropdown && (
              <div className="filter-dropdown">
                {MAP_CONFIGS.filter(m => availableMaps.length === 0 || availableMaps.includes(m.id)).map(map => (
                  <button
                    key={map.id}
                    className={`dropdown-option ${filters.map === map.id ? 'selected' : ''}`}
                    onClick={() => handleMapSelect(map.id)}
                  >
                    <div 
                      className="map-color-dot" 
                      style={{ background: map.color }}
                    />
                    <span>{map.name}</span>
                    {filters.map === map.id && <Check size={14} />}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Date Filter */}
          <div className="filter-dropdown-wrapper">
            <button 
              className={`filter-btn ${filters.dateRange ? 'active' : ''}`}
              onClick={() => {
                setShowDateDropdown(!showDateDropdown);
                setShowMapDropdown(false);
                setShowResultDropdown(false);
              }}
            >
              <Calendar size={16} />
              <span>Fecha</span>
              <ChevronDown size={14} className={showDateDropdown ? 'rotated' : ''} />
            </button>
            
            {showDateDropdown && (
              <div className="filter-dropdown">
                {['Hoy', 'Esta semana', 'Este mes', 'Últimos 3 meses'].map(range => (
                  <button
                    key={range}
                    className={`dropdown-option ${filters.dateRange === range ? 'selected' : ''}`}
                    onClick={() => handleDateRangeSelect(range)}
                  >
                    <span>{range}</span>
                    {filters.dateRange === range && <Check size={14} />}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Result Filter */}
          <div className="filter-dropdown-wrapper">
            <button 
              className={`filter-btn ${filters.result ? 'active' : ''}`}
              onClick={() => {
                setShowResultDropdown(!showResultDropdown);
                setShowMapDropdown(false);
                setShowDateDropdown(false);
              }}
            >
              <Filter size={16} />
              <span>Resultado</span>
              <ChevronDown size={14} className={showResultDropdown ? 'rotated' : ''} />
            </button>
            
            {showResultDropdown && (
              <div className="filter-dropdown">
                <button
                  className={`dropdown-option ${filters.result === 'victory' ? 'selected' : ''}`}
                  onClick={() => handleResultSelect('victory')}
                >
                  <div className="result-indicator victory" />
                  <span>Victoria</span>
                  {filters.result === 'victory' && <Check size={14} />}
                </button>
                <button
                  className={`dropdown-option ${filters.result === 'defeat' ? 'selected' : ''}`}
                  onClick={() => handleResultSelect('defeat')}
                >
                  <div className="result-indicator defeat" />
                  <span>Derrota</span>
                  {filters.result === 'defeat' && <Check size={14} />}
                </button>
              </div>
            )}
          </div>
        </div>

        {/* Spacer */}
        <div className="filters-spacer" />

        {/* View Toggle */}
        <div className="view-toggle">
          <button 
            className={`view-btn ${currentView === 'cards' ? 'active' : ''}`}
            onClick={() => onViewChange('cards')}
            title="Vista de cards"
          >
            <LayoutGrid size={18} />
          </button>
          <button 
            className={`view-btn ${currentView === 'table' ? 'active' : ''}`}
            onClick={() => onViewChange('table')}
            title="Vista de tabla"
          >
            <List size={18} />
          </button>
        </div>
      </div>

      {/* Active Pills */}
      {activePills.length > 0 && (
        <div className="active-pills">
          {activePills.map(pill => (
            <div 
              key={pill.id} 
              className="filter-pill"
              style={pill.color ? { borderColor: pill.color } : {}}
            >
              {pill.color && (
                <div 
                  className="pill-dot" 
                  style={{ background: pill.color }}
                />
              )}
              <span>{pill.label}</span>
              <button 
                className="pill-remove"
                onClick={() => removePill(pill.id)}
              >
                <X size={12} />
              </button>
            </div>
          ))}
          <button className="clear-all-btn" onClick={clearAllFilters}>
            Limpiar filtros
          </button>
        </div>
      )}
    </div>
  );
};

export default MatchFilters;
