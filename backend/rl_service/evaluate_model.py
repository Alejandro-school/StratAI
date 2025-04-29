from stable_baselines3 import PPO
from environment import CS2Env

def evaluate():
    """Evalúa el modelo RL entrenado."""
    env = CS2Env()
    model = PPO.load("cs2_coach_rl_model")

    obs = env.reset()
    total_reward = 0
    for _ in range(1000):
        action, _states = model.predict(obs)
        obs, reward, done, info = env.step(action)
        total_reward += reward
        if done:
            obs = env.reset()

    print(f"Recompensa total obtenida tras evaluación: {total_reward}")

if __name__ == "__main__":
    evaluate()
