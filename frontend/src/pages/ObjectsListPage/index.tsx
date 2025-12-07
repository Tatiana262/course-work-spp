import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { fetchObjects } from '@/features/RealEstateObjects/api/objectsApi';
import { ObjectCard } from '@/entities/RealEstateObject/ui/ObjectCard';
// import { FiltersPanel } from '@/features/Objects/ui/FiltersPanel';
// import { Pagination } from '@/shared/ui/Pagination';

export const ObjectsListPage = () => {
  const [filters, setFilters] = useState({ page: 1, perPage: 20 });
  
  // React Query для загрузки данных об объектах
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['objects', filters], // Ключ кэша зависит от фильтров!
    queryFn: () => fetchObjects(filters),
  });

  if (isLoading) return <div>Загрузка объектов...</div>;
  if (isError) return <div>Ошибка: {error.message}</div>;

  return (
    <div>
      <h1>Каталог недвижимости</h1>
      {/* <FiltersPanel onFilterChange={setFilters} /> */}
      
      <div className="objects-grid">
        {data?.data.map(obj => (
          <ObjectCard key={obj.id} object={obj} />
        ))}
      </div>
      
      {/* <Pagination 
        currentPage={data?.page}
        totalPages={Math.ceil(data?.total / data?.perPage)}
        onPageChange={(page) => setFilters(prev => ({ ...prev, page }))}
      /> */}
    </div>
  );
};