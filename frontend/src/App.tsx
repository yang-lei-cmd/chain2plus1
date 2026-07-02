import { useEffect } from 'react';
import { useAuth } from './lib/auth';
import { ToastProvider } from './components/Toast';
import Layout from './components/Layout';
import LoginPage from './pages/LoginPage';
import HomePage from './pages/HomePage';
import OrderPage from './pages/OrderPage';
import WithdrawPage from './pages/WithdrawPage';
import ProfitPage from './pages/ProfitPage';
import ProfilePage from './pages/ProfilePage';
import FreelancePage from './pages/FreelancePage';
import SharePage from './pages/SharePage';
import AdminPage from './pages/AdminPage';
import { useToast } from './components/Toast';
import './styles.css';

function AppContent() {
  const { isLoggedIn } = useAuth();
  const { show } = useToast();
  const hash = window.location.hash.slice(1) || '/';

  // Handle 401 redirect
  if (!isLoggedIn && hash !== '/login') {
    window.location.hash = '/login';
    return <LoginPage />;
  }
  if (!isLoggedIn) {
    return <LoginPage />;
  }

  // Bind hash change for SPA navigation
  useEffect(() => {
    const handler = () => {
      // Force re-render on hash change
      window.dispatchEvent(new Event('hashchange-force'));
    };
    window.addEventListener('hashchange', handler);
    return () => window.removeEventListener('hashchange', handler);
  }, []);

  const renderPage = () => {
    switch (hash) {
      case '/': return <HomePage />;
      case '/orders': return <OrderPage />;
      case '/withdraw': return <WithdrawPage />;
      case '/profits': return <ProfitPage />;
      case '/profile': return <ProfilePage />;
      case '/freelance': return <FreelancePage />;
      case '/share': return <SharePage />;
      case '/admin': return <AdminPage />;
      default: return <HomePage />;
    }
  };

  return <Layout>{renderPage()}</Layout>;
}

export default function App() {
  return (
    <ToastProvider>
      <AppContent />
    </ToastProvider>
  );
}
