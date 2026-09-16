import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import toast from 'react-hot-toast';
import { Sparkles } from 'lucide-react';

import { searchTables } from '../api/search';
import { DEFAULT_PAGE_SIZE } from '../utils/constants';
import { FilterPanel } from '../components/search/FilterPanel';
import { TableAvailabilityCard } from '../components/search/TableAvailabilityCard';
import { Pagination } from '../components/search/Pagination';
import { Loader } from '../components/common/Loader';
import { ErrorMessage } from '../components/common/ErrorMessage';
import { useAuth } from '../hooks/useAuth';
import { useCreateReservation } from '../hooks/useReservations';

const highlightCards = [
  { label: 'Búsqueda', value: 'Apache Solr', description: 'Índice dedicado, separado de la base transaccional' },
  { label: 'Sincronización', value: 'RabbitMQ', description: 'Eventos asíncronos entre reservas y búsqueda' },
  { label: 'Caché', value: '2 capas', description: 'Local + Memcached, con invalidación por evento' },
];

const Home = () => {
  const navigate = useNavigate();
  const { isAuthenticated, user } = useAuth();
  const createReservationMutation = useCreateReservation();
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState({ meal_type: '', is_available: '', capacity: '', date: '' });

  const params = useMemo(
    () => ({
      q: '*:*',
      page,
      size: DEFAULT_PAGE_SIZE,
      ...filters,
    }),
    [filters, page],
  );

  const { data, isLoading, isError, refetch, isFetching } = useQuery({
    queryKey: ['search-tables', params],
    queryFn: () => searchTables(params),
    staleTime: 1000 * 30,
    refetchInterval: 1000 * 5,
    refetchOnReconnect: true,
    refetchOnWindowFocus: true,
  });

  const handleFilterChange = (nextFilters) => {
    setFilters(nextFilters);
    setPage(1);
  };

  const results = data?.Results ?? [];

  const handleQuickReserve = async (table) => {
    if (!isAuthenticated || !user?.id) {
      toast.error('Iniciá sesión para reservar una mesa.');
      navigate('/login');
      return;
    }

    const dateTime = new Date(`${table.date}T12:00:00`);

    const payload = {
      owner_id: String(user.id),
      table_number: table.table_number,
      guests: table.capacity || 1,
      meal_type: table.meal_type,
      date_time: dateTime.toISOString(),
    };

    try {
      await createReservationMutation.mutateAsync(payload);
      refetch();
    } catch {
      // Not needed: hook already muestra toast
    }
  };

  return (
    <div className="mx-auto max-w-6xl space-y-10 px-4 py-12 text-white">
      <section className="relative overflow-hidden rounded-[46px] border border-white/10 bg-gradient-to-br from-[#0b1427] via-[#193364] to-[#2563eb] p-10 shadow-[0_40px_80px_-30px_rgba(0,0,0,0.9)]">
        <div className="absolute inset-0 opacity-70 blur-3xl" style={{ background: 'radial-gradient(circle at 10% 20%, rgba(255,255,255,0.25), transparent 45%)' }} />
        <div className="relative">
          <p className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.4em] text-white/70">
            <Sparkles size={16} />
            Mesas disponibles
          </p>
          <h1 className="mt-4 font-display text-4xl font-light leading-tight tracking-tight sm:text-[3.4rem]">
            Encontrá tu mesa ideal
          </h1>
          <p className="mt-4 max-w-2xl text-base text-white/80 sm:text-lg">
            Consulta disponibilidad en tiempo real, reservá tu mesa y controlá cada detalle con un panel intuitivo potenciado por automatizaciones.
          </p>
          <div className="mt-8 grid gap-4 sm:grid-cols-3">
            {highlightCards.map((card) => (
              <article key={card.label} className="rounded-3xl border border-white/15 bg-white/5 p-6 text-center backdrop-blur-2xl shadow-lg shadow-black/30">
                <p className="text-[0.55rem] uppercase tracking-[0.5em] text-white/60">{card.label}</p>
                <p className="mt-2 font-display text-3xl font-light">{card.value}</p>
                <p className="mt-1 text-xs text-white/70">{card.description}</p>
              </article>
            ))}
          </div>
        </div>
      </section>

      <div className="space-y-4">
        <FilterPanel filters={filters} onChange={handleFilterChange} visible={true} />
      </div>

      {isLoading || isFetching ? (
        <Loader label="Buscando mesas disponibles..." />
      ) : isError ? (
        <ErrorMessage message="No pudimos obtener los resultados" actionLabel="Reintentar" onAction={() => refetch()} />
      ) : (
        <>
          <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {results.length ? (
              results.map((table) => (
                <TableAvailabilityCard key={table.id} table={table} onReserve={handleQuickReserve} />
              ))
            ) : (
              <div className="rounded-3xl border border-dashed border-white/20 px-6 py-10 text-center text-sm text-slate-300 backdrop-blur-lg sm:col-span-2 lg:col-span-3">
                <p>Sin mesas disponibles. Refiná tu búsqueda o ajustá los filtros.</p>
              </div>
            )}
          </div>
          <Pagination page={data?.Page ?? 1} pages={data?.Pages ?? 1} onChange={setPage} />
        </>
      )}

      {!isAuthenticated && (
        <div className="rounded-[32px] border border-white/15 bg-white/5 p-8 text-center shadow-2xl shadow-black/40 backdrop-blur-2xl">
          <p className="text-2xl font-semibold">¿Querés confirmar una reserva?</p>
          <p className="mt-3 text-sm text-slate-200">
            Creá una cuenta o iniciá sesión para acceder a tus reservas y panel de administración inteligente.
          </p>
        </div>
      )}
    </div>
  );
};

export default Home;
