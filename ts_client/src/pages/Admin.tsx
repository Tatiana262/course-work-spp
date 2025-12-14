import React, { useState } from 'react';
import { Container, Tab, Tabs } from 'react-bootstrap';
import AdminControls from '../components/admin/AdminControls';
import TaskMonitor from '../components/admin/TaskMonitor';

const Admin = () => {
    // Состояние активной вкладки. По умолчанию открываем "Управление".
    const [key, setKey] = useState('controls');

    return (
        <Container className="mt-4">
            <div className="d-flex justify-content-between align-items-center mb-4">
                <h2>Панель администратора</h2>
                {/* Здесь можно добавить какую-нибудь статистику или кнопку "Выход из админки" */}
            </div>
            
            <Tabs
                id="admin-tabs"
                activeKey={key}
                onSelect={(k) => setKey(k || 'controls')}
                className="mb-3"
                // mountOnEnter={true} // Рендерить вкладку только при первом открытии (оптимизация)
                unmountOnExit={true} // Не удалять DOM при переключении (чтобы не рвать SSE соединение)
            >
                {/* --- ВКЛАДКА 1: ЗАПУСК ЗАДАЧ --- */}
                <Tab eventKey="controls" title="Управление парсерами">
                    <div className="mt-3">
                        <AdminControls />
                    </div>
                </Tab>

                {/* --- ВКЛАДКА 2: СПИСОК ЗАДАЧ --- */}
                <Tab eventKey="tasks" title="Мониторинг задач (Real-time)">
                    <div className="mt-3">
                        <TaskMonitor />
                    </div>
                </Tab>
            </Tabs>
        </Container>
    );
};

export default Admin;