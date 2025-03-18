import React from 'react';
import SteamLoginButton from '../../auth/SteamLoginButton.jsx';
import BodyVideo from '../Layout/BodyVideo.jsx';
import '../../styles/Start/landing.css';
import { motion } from 'framer-motion';
import { Box } from '@mui/material';

const LandingPage = () => {
  return (
    
    <div className="landing-container">

    <BodyVideo/>

      {/* Contenido centrado con efecto de vidrio esmerilado */}
      <motion.div

          initial={{ opacity: 0, y: '-100vh'}}// Inicia completamente fuera de la pantalla
          animate={{ opacity: 1, y: '0'}} // Se desplaza suavemente al centro
          transition={{ duration: 1.5, ease: 'easeOut' }}  // Mayor duración para suavizar
      >

      
        <Box
          sx={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            backdropFilter: 'blur(10px)',
            backgroundColor: 'rgba(255, 255, 255, 0.1)',
            padding: '50px',
            borderRadius: '15px',
            boxShadow: '0 4px 30px rgba(0, 0, 0, 0.1)',
            color: '#fff',
            maxWidth: '800px',
            width: '100%',
          }}
        >
          <h1 className="landing-title">STRATAI</h1>
          <p className="landing-subtitle">
            Domina tus partidas con estadísticas avanzadas y análisis detallados.
          </p>
          <br></br>
          <SteamLoginButton />
        </Box>
      </motion.div>
    </div>
  );
};

export default LandingPage;
