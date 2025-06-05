import React from 'react';
import videoBackground from '../../media/Background.mp4';
import '../../styles/Start/dashboard.css';

const BodyVideo = () => (
  <>
    {/* contenedor para aplicar gradiente sobre el video */}
    <div className="video-wrapper">
      <div className="video-gradient" />
      {/* vídeo fijo a pantalla completa */}
      <video
        className="background-video"
        src={videoBackground}
        autoPlay
        loop
        muted
        playsInline   // evita pantallazo en iOS
      />
    </div>

    {/*  oscurecido opcional; ajusta el 0.35 a tu gusto  */}
    <div className="video-overlay" />
  </>
);

export default BodyVideo;
