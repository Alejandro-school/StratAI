import React from 'react';
import NavigationFrame from '../components/Layout/NavigationFrame';
import { useAuth } from '../auth/useAuth';
import OperationProgress from '../features/progress/OperationProgress';

const Progress = () => {
  const { user } = useAuth();

  return (
    <NavigationFrame>
      <OperationProgress userName={user?.username || 'Tu perfil'} />
    </NavigationFrame>
  );
};

export default Progress;
