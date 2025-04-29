import React from 'react';
import '../../styles/Start/dashboard.css';
import videoBackground from '../../media/Background.mp4'; // Ruta del video de fondo

const BodyVideo = () => {
  return (
    <div className="video-container">
      <video autoPlay loop muted className="background-video">
        <source src={videoBackground} type="video/mp4" />
      </video>
    </div>
  );
};

export default BodyVideo;
