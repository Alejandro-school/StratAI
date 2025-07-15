// Archivo: Footer.jsx
import React from 'react';
import styles from '../../styles/Landing/Footer.module.css'; 

const Footer = () => {
  return (
    <footer className={styles.footer}>
      <div className={styles.container}>
        <p>© {new Date().getFullYear()} StratAI. Todos los derechos reservados.</p>
        <ul className={styles.links}>
          <li><a href="#">Términos</a></li>
          <li><a href="#">Privacidad</a></li>
          <li><a href="#">Contacto</a></li>
        </ul>
      </div>
    </footer>
  );
};

export default Footer;
