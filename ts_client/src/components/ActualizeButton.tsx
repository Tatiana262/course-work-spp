import React, { useContext } from 'react';
import { Button, Spinner, OverlayTrigger, Tooltip } from 'react-bootstrap';
import { actualizeObject } from '../http/adminAPI';
import actualizationStore from '../store/ActualizationStore';
import { observer } from 'mobx-react-lite';
import { Context } from '../main';

interface Props {
    master_object_id: string;
    variant?: string; // Чтобы менять стиль (для детальной и для карточки)
    size?: 'sm' | 'lg';
}

const ActualizeButton: React.FC<Props> = observer(({ master_object_id, variant = "outline-primary", size = "sm" }) => {
    const { user, actualization } = useContext(Context);
    
    // Смотрим в стор: крутится ли сейчас этот объект?
    const isProcessing = actualization.isProcessing(master_object_id);

    const handleClick = async (e: React.MouseEvent) => {
        e.stopPropagation();
        e.preventDefault();

        if (!user.isAuth) return;

        // 1. Сразу ставим спиннер
        actualization.add(master_object_id);

        try {
            await actualizeObject(master_object_id);
            // Спиннер НЕ убираем. Он уберется сам, когда придет SSE событие в App.tsx
        } catch (e) {
            console.error(e);
            alert("Ошибка при запуске актуализации");
            actualization.remove(master_object_id); // При ошибке убираем сразу
        }
    };

    if (!user.isAuth) return null;

    return (
        <OverlayTrigger
            placement="top"
            overlay={<Tooltip id={`tooltip-${master_object_id}`}>Обновить информацию об объекте</Tooltip>}
        >
            <Button 
                variant={variant}
                size={size}
                onClick={handleClick}
                disabled={isProcessing}
                className="d-flex align-items-center gap-2"
            >
                {isProcessing ? (
                    <>
                        <Spinner as="span" animation="border" size="sm" role="status" aria-hidden="true" />
                        <span>Обновление...</span>
                    </>
                ) : (
                    <>
                        <i className="bi bi-arrow-repeat"></i>
                        <span className="d-none d-md-inline">Обновить</span>
                    </>
                )}
            </Button>
        </OverlayTrigger>
    );
});

export default ActualizeButton;