import React, { useEffect, useMemo, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import NavigationFrame from '../components/Layout/NavigationFrame';
import { useUser } from '../context/UserContext';
import { Target, Trophy } from 'lucide-react';
import { leaderboardSeed, missionSeed, rewardSeed } from '../mocks/progressData';
import ProgressHeader from '../components/Progress/ProgressHeader';
import MissionsTab from '../components/Progress/MissionsTab';
import RankingTab from '../components/Progress/RankingTab';
import '../styles/Progress/progressShell.css';

const TABS = [
  { id: 'missions',  label: 'Misiones',      icon: Target },
  { id: 'ranking',   label: 'Ranking',        icon: Trophy },
];

const getTimeUntilMonthReset = () => {
  const now = new Date();
  const resetAt = new Date(now.getFullYear(), now.getMonth() + 1, 1, 0, 0, 0, 0);
  const delta = Math.max(resetAt.getTime() - now.getTime(), 0);
  return {
    days: Math.floor(delta / (1000 * 60 * 60 * 24)),
    hours: Math.floor((delta / (1000 * 60 * 60)) % 24),
    minutes: Math.floor((delta / (1000 * 60)) % 60),
  };
};

const tabContentVariants = {
  initial: { opacity: 0, y: 10 },
  animate: { opacity: 1, y: 0, transition: { duration: 0.3, ease: 'easeOut' } },
  exit:    { opacity: 0, y: -8, transition: { duration: 0.18 } },
};

const Progress = () => {
  const { user } = useUser();
  const [activeTab, setActiveTab] = useState('missions');
  const [monthReset, setMonthReset] = useState(() => getTimeUntilMonthReset());
  const [refreshCountdown, setRefreshCountdown] = useState(60);
  const [lastRefreshAt, setLastRefreshAt] = useState(() => new Date());
  const [isRefreshing, setIsRefreshing] = useState(false);

  useEffect(() => {
    const interval = setInterval(() => {
      setMonthReset(getTimeUntilMonthReset());
      setRefreshCountdown((v) => {
        if (v <= 1) {
          setLastRefreshAt(new Date());
          setIsRefreshing(true);
          return 60;
        }
        return v - 1;
      });
    }, 1000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (!isRefreshing) return;
    const t = setTimeout(() => setIsRefreshing(false), 1250);
    return () => clearTimeout(t);
  }, [isRefreshing]);

  const leaderboard = useMemo(() => {
    const name = user?.username || 'Tu perfil';
    return leaderboardSeed.map((e) => ({ ...e, name: e.isCurrentUser ? name : e.name }));
  }, [user?.username]);

  const currentPlayer = leaderboard.find((e) => e.isCurrentUser) || leaderboard[0];

  return (
    <NavigationFrame>
      <div className="ps">
        {/* Background texture */}
        <div className="ps__bg" />

        {/* Header strip */}
        <ProgressHeader
          currentPlayer={currentPlayer}
          monthReset={monthReset}
          refreshCountdown={refreshCountdown}
          lastRefreshAt={lastRefreshAt}
          isRefreshing={isRefreshing}
        />

        {/* Tab bar */}
        <nav className="ps__tabs">
          {TABS.map((tab) => {
            const Icon = tab.icon;
            const isActive = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                className={`ps__tab ${isActive ? 'ps__tab--active' : ''}`}
                onClick={() => setActiveTab(tab.id)}
              >
                <Icon size={14} />
                <span>{tab.label}</span>
                {isActive && (
                  <motion.div
                    className="ps__tab-indicator"
                    layoutId="tabIndicator"
                    transition={{ type: 'spring', stiffness: 420, damping: 32 }}
                  />
                )}
              </button>
            );
          })}
        </nav>

        {/* Tab content */}
        <div className="ps__content">
          <AnimatePresence mode="wait">
            {activeTab === 'missions' && (
              <motion.div key="missions" className="ps__panel" {...tabContentVariants}>
                <MissionsTab missions={missionSeed} />
              </motion.div>
            )}
            {activeTab === 'ranking' && (
              <motion.div key="ranking" className="ps__panel" {...tabContentVariants}>
                <RankingTab leaderboard={leaderboard} currentPlayer={currentPlayer} rewards={rewardSeed} />
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>
    </NavigationFrame>
  );
};

export default Progress;
