import gym
import numpy as np
import json
import os
from gym import spaces

class CS2Env(gym.Env):
    """
    Entorno personalizado para entrenar un modelo RL en Counter-Strike 2.
    Usa datos generados a partir del análisis de demos en formato JSON.
    """

    def __init__(self, data_path='../data/exports/'):
        super(CS2Env, self).__init__()
        
        self.data_path = data_path
        self.matches = self._load_matches()
        self.current_match = None
        self.current_step = 0
        
        # Define un espacio de observación con valores normalizados (ajustar según necesidades)
        self.observation_space = spaces.Box(low=0, high=1, shape=(20,), dtype=np.float32)
        
        # Espacio de acciones (puedes ajustar el número de acciones según tu caso específico)
        self.action_space = spaces.Discrete(5)

    def _load_matches(self):
        """Carga los datos de las partidas analizadas."""
        matches = []
        for match_id in os.listdir(self.data_path):
            player_stats_path = os.path.join(self.data_path, match_id, 'player_stats.csv')
            if os.path.exists(player_stats_path):
                with open(player_stats_path, 'r') as f:
                    matches.append(f.readlines())
        return matches

    def reset(self):
        """Reinicia el entorno para empezar un nuevo episodio."""
        self.current_match = np.random.choice(self.matches)
        self.current_step = 1  # Evitar cabecera CSV
        return self._get_observation()

    def step(self, action):
        """Ejecuta un paso en el entorno."""
        done = False
        reward = self._calculate_reward(action)

        self.current_step += 1
        if self.current_step >= len(self.current_match) - 1:
            done = True

        observation = self._get_observation()
        info = {}

        return observation, reward, done, info

    def _get_observation(self):
        """Obtiene observación actual del entorno."""
        line = self.current_match[self.current_step].strip().split(',')
        stats = np.array(line[4:24]).astype(np.float32)  # Ajusta según columnas útiles
        stats = stats / (np.max(stats) + 1e-5)  # Normalización simple

        return stats

    def _calculate_reward(self, action):
        """Calcula la recompensa basada en la acción tomada."""
        # Ejemplo sencillo: recompensa basada en kills y muertes
        line = self.current_match[self.current_step].strip().split(',')
        kills = int(line[4])
        deaths = int(line[6])

        reward = kills - deaths  # Ajusta esta lógica según necesidades

        return reward
