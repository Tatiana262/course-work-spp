import React, { useContext, useEffect, useState } from 'react';
import { Container, Row, Col, Carousel, Card, Badge, Table, Spinner, Button, Alert, ListGroup } from 'react-bootstrap';
import { useParams, useNavigate } from 'react-router-dom';
import { fetchObjectWithDeatils } from '../http/objectsAPI';
import type { IObjectDetailsResponse, IApartmentDetails, IHouseDetails, IRelatedOffer, ICommercialDetails } from '../types/realEstateObjects';
import FavoriteButton from '../components/FavoriteButton';
import ActualizeButton from '../components/ActualizeButton';
import { Context } from '../main';
import { observer } from 'mobx-react-lite';

const ObjectPage = observer(() => {
    const { actualization } = useContext(Context);
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    
    const [data, setData] = useState<IObjectDetailsResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [isDescExpanded, setIsDescExpanded] = useState(false);

    const lastUpdateTimestamp = id ? actualization.updates.get(id) : undefined;

    const loadData = () => {
        if (!id) return;
        setLoading(true); // Можно сделать мягкую загрузку (без спиннера на весь экран), если хотите
        fetchObjectWithDeatils(id)
            .then(setData)
            .catch(err => setError('Ошибка'))
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        loadData();
    }, [id]);

    useEffect(() => {
        if (lastUpdateTimestamp) {
            // Если пришел сигнал - обновляем данные "тихо" (или с лоадером, как решите)
            console.log("Получен сигнал обновления, перезагружаем данные...");
            
            // Важно: fetchOneProperty возвращает свежие данные
            fetchObjectWithDeatils(id!).then(newData => {
                setData(newData);
                // Можно показать тост: "Цена обновлена!"
            });
        }
    }, [lastUpdateTimestamp]);

    if (loading) return <Container className="mt-5 text-center"><Spinner animation="border" variant="primary" /></Container>;
    if (error || !data) return <Container className="mt-5"><Alert variant="danger">{error || "Ошибка загрузки данных"}</Alert></Container>;

    const { general, details, related_offers } = data;

    const currencyMap: Record<string, string> = {
        'USD': '$',
        'EUR': '€',
        'BYN': 'BYN',
        'RUB': '₽'
    };

    const currencySymbol = currencyMap[general.currency] || general.currency;

    const descriptionLength = general.description?.length || 0;
    const isLongDescription = descriptionLength > 500;


    const category = general.category; // 'apartment', 'house', 'commercial'

    const isApartment = category === 'apartment';
    const isHouse = category === 'house';
    const isCommercial = category === 'commercial';

    // Приведение типов (Casting)
    const aptDetails = isApartment ? (details as IApartmentDetails) : null;
    const houseDetails = isHouse ? (details as IHouseDetails) : null;
    const commDetails = isCommercial ? (details as ICommercialDetails) : null;

    // --- ЛОГИКА ГРУППИРОВКИ ДУБЛИКАТОВ ---
    // Группируем массив related_offers по полю source
    const groupedOffers = related_offers.reduce((acc, offer) => {
        const src = offer.source || 'other';
        if (!acc[src]) {
            acc[src] = [];
        }
        acc[src].push(offer);
        return acc;
    }, {} as Record<string, IRelatedOffer[]>);

    const sources = Object.keys(groupedOffers);

    // --- ХЕЛПЕР РЕНДЕРА СТРОКИ ---
    const renderRow = (label: string, value: any, suffix = '') => {
        if (value === null || value === undefined || value === '') return null;
        if (value === 0 && suffix === '') return null; // Иногда 0 стоит скрывать, но зависит от контекста

        let displayValue = value;
        if (typeof value === 'boolean') displayValue = value ? 'Да' : 'Нет';

        return (
            <tr key={label}>
                <td className="text-muted w-50">{label}</td>
                <td>{displayValue} {suffix}</td>
            </tr>
        );
    };

    const renderRoomsRange = (label: string, range?: number[]) => {
        if (!range || range.length === 0) return null;
        
        let value = '';
        if (range.length === 1) {
            value = `${range[0]}`;
        } else {
            // Если два числа - выводим через тире
            value = range.join(' - ');
        }
        
        return (
            <tr key={label}>
                <td className="text-muted w-50">{label}</td>
                <td>{value}</td>
            </tr>
        );
    };

    return (
        <Container className="mt-4 mb-5">
            <Button variant="outline-secondary" className="mb-3" onClick={() => navigate(-1)}>
                &larr; Назад к списку
            </Button>

            <Row>
                {/* === ЛЕВАЯ КОЛОНКА (Фото, Описание, Детали) === */}
                <Col lg={8}>
                    
                    {/* 1. ФОТОГАЛЕРЕЯ */}
                    <Card className="mb-4 shadow-sm overflow-hidden border-0">
                         {general.images && general.images.length > 0 ? (
                            <Carousel>
                                {general.images.map((img, index) => (
                                    <Carousel.Item key={index} style={{ height: '500px', background: '#222' }}>
                                        <img 
                                            className="d-block w-100 h-100" 
                                            src={img} 
                                            alt={`Фото ${index + 1}`} 
                                            style={{ objectFit: 'contain' }} 
                                        />
                                    </Carousel.Item>
                                ))}
                            </Carousel>
                        ) : (
                            <div className="bg-light d-flex align-items-center justify-content-center text-muted" style={{height: '400px'}}>
                                Фотографии отсутствуют
                            </div>
                        )}
                    </Card>

                    {/* 2. ОПИСАНИЕ */}
                    <Card className="mb-4 shadow-sm p-4 border-0">
                        <h4 className="mb-3">Описание</h4>
                        
                        <div style={{ 
                            // Если свернуто и текст длинный — ограничиваем высоту
                            maxHeight: (!isDescExpanded && isLongDescription) ? '200px' : 'none', 
                            overflow: 'hidden',
                            position: 'relative',
                            transition: 'max-height 0.3s ease'
                        }}>
                            {general.description?.includes('<') ? (
                                <div dangerouslySetInnerHTML={{ __html: general.description }} />
                            ) : (
                                <p style={{ whiteSpace: 'pre-wrap' }}>{general.description || "Описание отсутствует."}</p>
                            )}
                            
                            {/* Эффект затемнения внизу, если текст свернут */}
                            {!isDescExpanded && isLongDescription && (
                                <div style={{
                                    position: 'absolute',
                                    bottom: 0,
                                    left: 0,
                                    width: '100%',
                                    height: '60px',
                                    background: 'linear-gradient(transparent, white)'
                                }} />
                            )}
                        </div>

                        {/* Кнопка Раскрыть / Свернуть */}
                        {isLongDescription && (
                            <div className="text-center mt-2">
                                <Button 
                                    variant="link" 
                                    className="text-decoration-none p-0"
                                    onClick={() => setIsDescExpanded(!isDescExpanded)}
                                >
                                    {isDescExpanded ? 'Свернуть описание' : 'Читать полностью'}
                                </Button>
                            </div>
                        )}
                    </Card>

                    {/* 3. ХАРАКТЕРИСТИКИ */}
                    <Card className="shadow-sm p-4 mb-4 border-0">
                        <h4 className="mb-3">Характеристики</h4>
                        <Table striped bordered hover size="sm">
                            <tbody>
                                {/* Общие */}
                               
                              
                                {/* ДЛЯ КВАРТИР */}
                                {isApartment && aptDetails && (
                                    <>
                                        {renderRow("Категория", translateCategory(general.category))}
                                        {renderRow("Год постройки", aptDetails.year_built)}
                                        {renderRow("Материал стен", aptDetails.wall_material)}
                                        
                                        {renderRow("Общая площадь", aptDetails.total_area, "м²")}
                                        {renderRow("Жилая площадь", aptDetails.living_space_area, "м²")}
                                        {renderRow("Кухня", aptDetails.kitchen_area, "м²")}
                                        
                                        {renderRow("Этаж", `${aptDetails.floor_number || '?'} из ${aptDetails.building_floors || '?'}`)}
                                        {renderRow("Комнат", aptDetails.rooms_amount)}
                                        {renderRow("Санузел", aptDetails.bathroom_type)}
                                        {renderRow("Балкон", aptDetails.balcony_type)}
                                        {renderRow("Ремонт", aptDetails.repair_state)}
                                        {renderRow("Состояние новое", aptDetails.is_new_condition)}
                                    </>
                                )}

                                {/* ДЛЯ ДОМОВ */}
                                {isHouse && houseDetails && (
                                    <>
                                        {renderRow("Категория", translateCategory(general.category))}
                                        {renderRow("Год постройки", houseDetails.year_built)}
                                        {renderRow("Материал стен", houseDetails.wall_material)}
                                        
                                        {renderRow("Общая площадь", houseDetails.total_area, "м²")}
                                        {renderRow("Жилая площадь", houseDetails.living_space_area, "м²")}
                                        {renderRow("Кухня", houseDetails.kitchen_area, "м²")}

                                        {renderRow("Тип объекта", houseDetails.house_type)}
                                        {renderRow("Участок", houseDetails.plot_area, "сот.")}
                                        {renderRow("Этажность", houseDetails.building_floors)}
                                        {renderRow("Комнат", houseDetails.rooms_amount)}
                                        
                                        {renderRow("Отопление", houseDetails.heating)}
                                        {renderRow("Вода", houseDetails.water)}
                                        {renderRow("Канализация", houseDetails.sewage)}
                                        {renderRow("Газ", houseDetails.gaz)}
                                        {renderRow("Электричество", houseDetails.electricity)}
                                        {renderRow("Крыша", houseDetails.roof_material)}
                                        {renderRow("Готовность", houseDetails.completion_percent, "%")}
                                        {renderRow("Состояние новое", houseDetails.is_new_condition)}
                                    </>
                                )}

                                {isCommercial && commDetails && (
                                    <>
                                        {renderRow("Категория", "Коммерческая недвижимость")}
                                        {renderRow("Вид объекта", commDetails.property_type)}
                                        {renderRow("Расположение", commDetails.commercial_building_location)}
                                                                                
                                        {renderRow("Общая площадь", commDetails.total_area, "м²")}
                                        
                                        {renderRow("Этаж", commDetails.building_floors 
                                            ? `${commDetails.floor_number || '?'} из ${commDetails.building_floors}` 
                                            : commDetails.floor_number
                                        )}

                                        {/* Вывод диапазона комнат (1 или 2-5) */}
                                        {renderRoomsRange("Кол-во помещений", commDetails.rooms_range)}

                                        {renderRow("Тип аренды", commDetails.commercial_rent_type)}
                                        {renderRow("Ремонт", commDetails.commercial_repair)}
                                        {renderRow("Состояние новое", commDetails.is_new_condition)}

                                        {commDetails.commercial_improvements && commDetails.commercial_improvements.length > 0 && (
                                            renderRow("Удобства", commDetails.commercial_improvements.join(', '))
                                        )}
                                    </>
                                )}
                            </tbody>
                        </Table>
                    </Card>

                    {/* 4. ДОПОЛНИТЕЛЬНЫЕ ПАРАМЕТРЫ (из JSONB parameters) */}
                    {Object.keys(details.parameters).length > 0 && (
                        <Card className="shadow-sm p-4 mb-4 border-0">
                            <h5 className="mb-3">Дополнительно</h5>
                            <Table size="sm" borderless>
                                <tbody>
                                    {Object.entries(details.parameters).map(([key, val]) => (
                                        // Рендерим только примитивы, пропуская вложенные объекты
                                        typeof val !== 'object' && val !== null && renderRow(translateParameter(key), val)
                                    ))}
                                </tbody>
                            </Table>
                        </Card>
                    )}

                    {/* 5. ПОХОЖИЕ ПРЕДЛОЖЕНИЯ (ГРУППИРОВКА) */}
                    {sources.length > 0 && (
                        <Card className="shadow-sm p-4 border-0">
                            <h4 className="mb-3">Этот объект на других сайтах</h4>
                            {sources.map(source => (
                                <div key={source} className="mb-3">
                                    <h6 className="text-muted text-uppercase fw-bold mt-2" style={{ fontSize: '0.8rem' }}>
                                        Найдено на {source}:
                                    </h6>
                                    <ListGroup variant="flush">
                                        {groupedOffers[source].map(offer => (
                                            <ListGroup.Item key={offer.id} className="d-flex justify-content-between align-items-center px-0 py-2">
                                                <div>
                                                    <span className="me-2">🔗</span>
                                                    {/* Можно добавить дату или цену, если они отличаются */}
                                                    {offer.is_source_duplicate && (
                                                        <Badge bg="warning" text="dark" className="me-2" style={{fontSize: '0.7em'}}>
                                                            Дубликат источника
                                                        </Badge>
                                                    )}
                                                    <span className="text-dark small">
                                                        {offer.deal_type === 'sale' ? 'Продажа' : 'Аренда'}
                                                    </span>
                                                </div>
                                                <Button 
                                                    variant="outline-primary" 
                                                    size="sm" 
                                                    href={offer.ad_link} 
                                                    target="_blank" 
                                                    rel="noreferrer"
                                                >
                                                    Перейти
                                                </Button>
                                            </ListGroup.Item>
                                        ))}
                                    </ListGroup>
                                </div>
                            ))}
                        </Card>
                    )}
                </Col>

                {/* === ПРАВАЯ КОЛОНКА (Цена, Адрес, Продавец) === */}
                <Col lg={4}>
                    <div className="sticky-top" style={{ top: '20px', zIndex: 10 }}>
                        {/* КАРТОЧКА ЦЕНЫ */}
                        <Card className="shadow-sm p-4 mb-3 border-0">
                            <h2 className="text-primary fw-bold">
                                {general.price_byn > 0 
                                    ? `${general.price_byn.toLocaleString('ru-RU')} BYN` 
                                    : "Договорная"
                                }
                            </h2>
                            {general.price_byn > 0 && (
                                <div className="d-flex gap-3 text-muted mb-3">
                                    <span>≈ {general.price_usd?.toLocaleString('ru-RU')} $</span>
                                    {general.price_eur && <span>≈ {general.price_eur?.toLocaleString('ru-RU')} €</span>}
                                </div>
                            )}
                            
                            {/* Цена за квадрат */}
                            {(isApartment && aptDetails?.price_per_square_meter) || 
                            (isCommercial && commDetails?.price_per_square_meter) ? (
                                <div className="mb-3 badge bg-light text-dark border p-2 fw-normal">
                                    {aptDetails?.price_per_square_meter || commDetails?.price_per_square_meter} {currencySymbol} / м²
                                </div>
                            ) : null}

                            <hr />

                            <div className="d-flex justify-content-between align-items-start mb-2">
                                <h5 className="me-2" style={{lineHeight: '1.4'}}>{general.title}</h5>
                                
                                {/* Кнопка Избранного */}
                                <div>
                                    <FavoriteButton masterObjectId={general.master_object_id} />
                                </div>
                            </div>
                            
                            <p className="text-muted mb-2">
                                <i className="bi bi-geo-alt-fill me-2 text-danger"></i>
                                {general.address}
                            </p>
                            
                            <div className="d-flex justify-content-between align-items-center mt-2">
                                <Badge bg={general.deal_type === 'sale' ? 'success' : 'info'} className="px-3 py-2">
                                    {general.deal_type === 'sale' ? 'Продажа' : 'Аренда'}
                                </Badge>
                                <span className="text-muted small">ID: {general.source_ad_id}</span>
                            </div>
                        </Card>

                        {/* КАРТОЧКА ПРОДАВЦА */}
                        <Card className="shadow-sm p-4 border-0">
                            <h5 className="mb-3">Продавец</h5>
                            
                            {general.seller_name && (
                                <h6 className="fw-bold mb-2">{general.seller_name}</h6>
                            )}

                            {/* Гибкий вывод деталей продавца */}
                            <div className="small text-muted">
                                {general.seller_details.contact_person && (
                                    <div className="mb-1">Контакт: {general.seller_details.contact_person}</div>
                                )}
                                
                                {general.seller_details.contactPhones && general.seller_details.contactPhones.length > 0 && (
                                    <div className="mb-2">
                                        {general.seller_details.contactPhones.map((ph: string) => (
                                            <div key={ph} className="fw-bold text-dark fs-6 my-1">
                                                <a href={`tel:${ph}`} className="text-decoration-none text-dark">{ph}</a>
                                            </div>
                                        ))}
                                    </div>
                                )}
                                
                                {general.seller_details.company_address && (
                                    <div className="mb-1">Адрес: {general.seller_details.company_address}</div>
                                )}

                                {/* Лицензии (разные форматы) */}
                                {general.seller_details.agency ? (
                                    <div className="mt-2 fst-italic border-top pt-2">
                                        Лицензия: {general.seller_details.agency.license} <br/>
                                        УНП: {general.seller_details.agency.unp}
                                    </div>
                                ) : general.seller_details.unp ? (
                                    <div className="mt-2 border-top pt-2">УНП: {general.seller_details.unp}</div>
                                ) : null}
                            </div>

                            <div className="mt-4">
                                <Button 
                                    variant="primary" 
                                    className="w-100 py-2" 
                                    href={general.ad_link} 
                                    target="_blank"
                                >
                                    Смотреть на {general.source}
                                </Button>
                                {general.list_time && (
                                   <div className="d-flex justify-content-between align-items-center mt-3">
                                        <div className="text-muted small">
                                            Размещено: {new Date(general.list_time).toLocaleString('ru-RU')}
                                        </div>
                                        <div className="text-muted small">
                                            Обновлено: {new Date(general.updated_at).toLocaleString('ru-RU')}
                                        </div>
                                        
                                        {/* Вставляем кнопку */}
                                        <ActualizeButton master_object_id={general.master_object_id} />
                                    </div>
                                )}
                            </div>
                        </Card>
                    </div>
                </Col>
            </Row>
        </Container>
    );
});

function translateCategory(cat: string) {
    const map: Record<string, string> = {
        'apartment': 'Квартира',
        'house': 'Дом, Коттедж',
        'room': 'Комната',
        'commercial': 'Коммерческая недвижимость',
        'plot': 'Участок',
        'garage': 'Гараж'
    };
    return map[cat] || cat;
}

function translateParameter(key: string) {
    const map: Record<string, string> = {
        // --- Общие и финансовые ---
       
        'flat_new_building': 'Новостройка',
        're_contract': 'Номер договора',
        'contract': 'Номер договора',
        'agency_contract': 'Договор с агентством',
        'is_price_haggle': 'Возможен торг',
        'possible_exchange': 'Возможен обмен',
        'is_auction': 'Аукцион',
        're_auction_sale': 'Аукцион',
        'installment_pro': 'Рассрочка',
        'flat_rent_prepayment': 'Предоплата за аренду',
        'leasePeriod': 'Срок аренды',
        'lease_period': 'Срок аренды',
        'vat': 'НДС',
        're_property_rights': 'Права на участок',

        // --- Характеристики здания и квартиры ---
        'flat_ceiling_height': 'Высота потолков',
        'ceiling_height': 'Высота потолков',
        'flat_storeys': 'Этажность дома',
        'separate_rooms': 'Раздельных комнат',
        'studio': 'Студия',
        'flat_open_room': 'Планировка open space',
        'size_snb': 'Площадь по СНБ, м²',
        'wall_material': 'Материал стен',
        'house_roof_material_type': 'Тип материала крыши',
        'year_built': 'Год постройки',

        // --- Удобства в квартире/доме ---
        'has_furniture': 'Мебель',
        'flat_furnished': 'Меблирована',
        'appliances': 'Бытовая техника',
        'flat_kitchen': 'Кухонная техника',
        'flat_bath': 'Оборудование в ванной',
        'has_fireplace': 'Камин',
        'has_pool': 'Бассейн',
        'has_bath': 'Баня / Сауна',

        // --- Удобства (Территория и парковка) ---
        'has_garage': 'Гараж',
        'has_parking_place': 'Парковочное место',
        'is_fenced_territory': 'Огороженная территория',
        'has_guest_house': 'Гостевой домик',
        're_outbuildings_size': 'Площадь хоз. построек, м²',

        // --- Коммуникации ---
        'electricity': 'Электричество',
        'gas': 'Газ',
        'water': 'Вода',
        're_hot_water': 'Горячая вода',
        'heating': 'Отопление',
        'sewage': 'Канализация',

        // --- Детали для аренды ---
        'flat_rent_couchettes': 'Спальных мест',
        'house_rent_couchettes': 'Спальных мест',
        'flat_rent_for_whom': 'Предпочтения по арендаторам',
        
        // --- Улучшения и прочее ---
        'flat_improvement': 'Благоустройство квартиры',
        'house_improvements': 'Благоустройство дома',
        'flat_building_improvements': 'Благоустройство здания',
        'has_signaling': 'Сигнализация',
        'has_video_intercom': 'Видеодомофон',
        'flat_windows_side': 'Окна выходят на',
        'views': 'Вид из окна',
        'content_video': 'Ссылка на видео',
        'trademark': 'Торговая марка',

        // --- Адресные компоненты ---
        'street_name': 'Улица',
        'town_district_name': 'Район города',
        'town_sub_district_name': 'Микрорайон',

        // --- Новостройки ---
        'new_buildings_apartment_complex': 'Жилой комплекс',

        // --- Коммерческая недвижимость ---
        'commercial_legal_address': 'Предоставление юр. адреса',
        'provides_legal_address': 'Предоставление юр. адреса',
        'commercial_pavilions_type': 'Тип павильона',
        'commercial_services_type': 'Тип услуг',
        'commercial_rent_workplace': 'Аренда рабочего места',
        're_special_purpose': 'Специальное назначение',
        
        // --- Площади (для домов) ---
        'house_sell_area': 'Продаваемая площадь, м²',
        'house_rent_area': 'Сдаваемая площадь, м²',
        'house_rent_near_area': 'Прилегающая территория, м²',
        'house_rent_services': 'Услуги'
    };
    return map[key] || key; // Если перевода нет, возвращаем ключ как есть (например, formatted_parameter)
}

export default ObjectPage;