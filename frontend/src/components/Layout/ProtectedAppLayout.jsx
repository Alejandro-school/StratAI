import { QueryClientProvider } from '@tanstack/react-query';
import { AuthProvider } from '../../auth/useAuth';
import RequireAuth from '../../auth/RequireAuth';
import queryClient from '../../lib/queryClient';

const ProtectedAppLayout = () => (
  <AuthProvider>
    <QueryClientProvider client={queryClient}>
      <RequireAuth />
    </QueryClientProvider>
  </AuthProvider>
);

export default ProtectedAppLayout;
