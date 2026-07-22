import React, { useState } from 'react';
import { MessageSquare } from 'lucide-react';
import NavigationFrame from '../components/Layout/NavigationFrame';
import { FeedbackForm, FeedbackHistory } from '../components/Feedback';
import '../styles/pages/feedback.css';

const Feedback = () => {
  const [refreshKey, setRefreshKey] = useState(0);

  const handleSubmitted = () => setRefreshKey((k) => k + 1);

  return (
    <NavigationFrame>
      <div className="fb-page">
        <header className="fb-header">
          <h1 className="fb-header-title">
            <MessageSquare size={22} className="fb-header-icon" />
            Comentarios
          </h1>
          <p className="fb-header-sub">
            Cuéntanos qué te parece la plataforma, reporta errores o propón mejoras.
          </p>
        </header>

        <FeedbackForm onSubmitted={handleSubmitted} />
        <FeedbackHistory refreshKey={refreshKey} />
      </div>
    </NavigationFrame>
  );
};

export default Feedback;
