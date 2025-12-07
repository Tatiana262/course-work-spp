import { ObjectCard as IObjectCard } from '@/shared/api/types';

interface Props {
  object: IObjectCard;
}

export const ObjectCard = ({ object }: Props) => (
  <div className="card">
    <img src={object.images?.[0]} alt={object.title} />
    <h3>{object.title}</h3>
    <p>{object.address}</p>
    <p>Цена: ${object.priceUSD}</p>
    {/* Здесь будет кнопка "В избранное" из features/Favorites */}
  </div>
);