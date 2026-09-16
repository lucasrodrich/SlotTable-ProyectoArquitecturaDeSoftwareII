import { Link, Navigate } from 'react-router-dom';
import { LoginForm } from '../components/auth/LoginForm';
import { useAuth } from '../hooks/useAuth';
import { APP_NAME } from '../utils/constants';

const Login = () => {
  const { isAuthenticated } = useAuth();

  if (isAuthenticated) {
    return <Navigate to="/" replace />;
  }

  return (
    <div className="mx-auto flex min-h-[80vh] max-w-5xl flex-col items-center justify-center gap-8 px-4 py-12">
      <div className="text-center">
        <p className="text-sm uppercase tracking-[0.3em] text-slate-400 dark:text-slate-500">Bienvenido</p>
        <h1 className="font-display text-4xl font-semibold text-white">Ingresa a {APP_NAME}</h1>
        <p className="mt-2 text-slate-500 dark:text-slate-400">Gestiona reservas, confirmaciones y el panel admin.</p>
      </div>
      <div className="w-full max-w-xl elegant-card p-8">
        <LoginForm />
        <p className="mt-4 text-center text-sm text-slate-500 dark:text-slate-400">
          ¿No tenés cuenta?{' '}
          <Link to="/register" className="font-semibold text-primary-600 dark:text-primary-400">
            Registrate acá
          </Link>
        </p>
      </div>
    </div>
  );
};

export default Login;
