from stable_baselines3 import PPO
from environment import CS2Env

def train():
    """Entrena el modelo RL usando PPO."""
    env = CS2Env()
    
    model = PPO('MlpPolicy', env, verbose=1, tensorboard_log="./tensorboard/")
    
    # Entrena por 100,000 pasos (ajusta según necesidades)
    model.learn(total_timesteps=100000)
    
    # Guarda el modelo entrenado
    model.save("cs2_coach_rl_model")
    print("Modelo entrenado y guardado correctamente.")

if __name__ == "__main__":
    train()
