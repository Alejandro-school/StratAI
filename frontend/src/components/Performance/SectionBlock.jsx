import React from 'react';

/**
 * SectionBlock — wraps a named section inside a tab panel.
 * Renders a subtle title divider + children.
 */
const SectionBlock = ({ title, count, children }) => (
  <div className="p-section">
    <div className="p-section-header">
      <h3 className="p-section-title">{title}</h3>
      {count != null && <span className="p-section-count">{count}</span>}
    </div>
    {children}
  </div>
);

export default SectionBlock;
