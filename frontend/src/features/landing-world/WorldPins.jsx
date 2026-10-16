import React from 'react';
import { FEATURES } from './journey';

export function WorldPins({ elements, enabled, selected, onSelect }) {
  return <div className="world-callouts" aria-label="Puntos del producto en Mirage">
    <svg className="world-leaders" aria-hidden="true">{FEATURES.map(feature => <line key={feature.id} ref={node => { (elements.current[feature.id] ||= {}).line = node; }} stroke={feature.color} />)}</svg>
    {FEATURES.map(feature => <button key={feature.id} ref={node => { (elements.current[feature.id] ||= {}).button = node; }}
      className="world-pin" style={{ '--pin-color': feature.color }} type="button"
      disabled={!enabled || (!!selected && selected !== feature.id)} aria-pressed={selected === feature.id}
      aria-label={`Explorar ${feature.label}`} onClick={() => onSelect(feature.id)}>
      <span className="world-pin-dot" /><span>{feature.label}</span><span className="world-pin-plus" aria-hidden="true">+</span>
    </button>)}
  </div>;
}
