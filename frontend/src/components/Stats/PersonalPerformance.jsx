import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { Box, Grid, Card, CardContent, Typography } from '@mui/material';
import { makeStyles } from '@mui/styles';
import SpeedIcon from '@mui/icons-material/Speed';
import WhatshotIcon from '@mui/icons-material/Whatshot';
import SportsEsportsIcon from '@mui/icons-material/SportsEsports';
import StarIcon from '@mui/icons-material/Star';
import { motion } from 'framer-motion';
import '../../styles/Stats/personalPerformance.css';

// Estilos con Material-UI (makeStyles)
const useStyles = makeStyles({
  card: {
    minWidth: 150,
    textAlign: 'center',
    padding: '1rem',
    margin: '1rem',
    backgroundColor: '#ffffff',
    boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
    borderRadius: '8px',
    transition: 'transform 0.2s ease-in-out',
    '&:hover': {
      transform: 'scale(1.05)',
    },
  },
  icon: {
    fontSize: '2.5rem',
    marginBottom: '0.5rem',
    color: '#3f51b5',
  },
  title: {
    fontWeight: 'bold',
    marginBottom: '0.5rem',
  },
  value: {
    fontSize: '1.5rem',
    color: '#333',
  },
});

const PersonalPerformanceDashboard = () => {
  const classes = useStyles();
  const [performance, setPerformance] = useState(null);
  const [loading, setLoading] = useState(true);

  // Llamada al backend para obtener el resumen de desempeño personal
  useEffect(() => {
    const fetchPerformance = async () => {
      try {
        const response = await axios.get('/api/performance'); // Ajusta la URL según tu endpoint
        setPerformance(response.data);
      } catch (error) {
        console.error('Error al obtener el desempeño personal:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchPerformance();
  }, []);

  if (loading) {
    return <Typography variant="h6">Cargando desempeño...</Typography>;
  }

  if (!performance) {
    return <Typography variant="h6">No se encontraron datos.</Typography>;
  }

  // Variantes para animación con Framer Motion
  const cardVariants = {
    hidden: { opacity: 0, y: 20 },
    visible: (i) => ({
      opacity: 1,
      y: 0,
      transition: { delay: i * 0.1 },
    }),
  };

  // Definición de las tarjetas a mostrar
  const cards = [
    {
      title: 'Partidas Jugadas',
      value: performance.matches_played,
      icon: <SpeedIcon className={classes.icon} />,
    },
    {
      title: 'K/D Ratio',
      value: performance.kd_ratio.toFixed(2),
      icon: <WhatshotIcon className={classes.icon} />,
    },
    {
      title: 'ADR Promedio',
      value: performance.average_adr.toFixed(1),
      icon: <SportsEsportsIcon className={classes.icon} />,
    },
    {
      title: 'Aim Placement',
      value: performance.average_aim_placement.toFixed(1) + '°',
      icon: <StarIcon className={classes.icon} />,
    },
    {
      title: 'Kills Totales',
      value: performance.total_kills,
      icon: <WhatshotIcon className={classes.icon} />,
    },
    {
      title: 'Muertes Totales',
      value: performance.total_deaths,
      icon: <WhatshotIcon className={classes.icon} />,
    },
    {
      title: 'Double Kills',
      value: performance.total_double_kills,
      icon: <StarIcon className={classes.icon} />,
    },
    {
      title: 'Triple Kills',
      value: performance.total_triple_kills,
      icon: <StarIcon className={classes.icon} />,
    },
    {
      title: 'Quad Kills',
      value: performance.total_quad_kills,
      icon: <StarIcon className={classes.icon} />,
    },
    {
      title: 'Aces',
      value: performance.total_aces,
      icon: <StarIcon className={classes.icon} />,
    },
    {
      title: 'Clutch Wins',
      value: performance.total_clutch_wins,
      icon: <StarIcon className={classes.icon} />,
    },
  ];

  return (
    <Box className="dashboard-container">
      <Typography variant="h4" align="center" gutterBottom>
        Desempeño Personal
      </Typography>
      <Grid container justifyContent="center">
        {cards.map((card, index) => (
          <Grid item xs={12} sm={6} md={4} key={card.title}>
            <motion.div
              custom={index}
              variants={cardVariants}
              initial="hidden"
              animate="visible"
            >
              <Card className={classes.card}>
                <CardContent>
                  {card.icon}
                  <Typography variant="h6" className={classes.title}>
                    {card.title}
                  </Typography>
                  <Typography variant="h5" className={classes.value}>
                    {card.value}
                  </Typography>
                </CardContent>
              </Card>
            </motion.div>
          </Grid>
        ))}
      </Grid>
    </Box>
  );
};

export default PersonalPerformanceDashboard;
