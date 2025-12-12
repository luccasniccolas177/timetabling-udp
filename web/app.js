const state = {
    scheduleData: null,
    allActivities: [],
    filteredActivities: [],
    filters: {
        day: '',
        course: '',
        teacher: '',
        room: '',
        type: '',
        search: ''
    },
    view: 'grid'
};

const DAYS = ['Lunes', 'Martes', 'Miércoles', 'Jueves', 'Viernes'];
const TIME_SLOTS = [
    { block: 0, time: '08:30-09:50' },
    { block: 1, time: '10:00-11:20' },
    { block: 2, time: '11:30-12:50' },
    { block: 3, time: '13:00-14:20' },
    { block: 4, time: '14:30-15:50' },
    { block: 5, time: '16:00-17:20' },
    { block: 6, time: '17:25-18:45' }
];
const BLOCKS_PER_DAY = 7;
const PROTECTED_BLOCK = 16; // Miércoles 11:30-12:50

const elements = {
    filterDay: document.getElementById('filter-day'),
    filterCourse: document.getElementById('filter-course'),
    filterTeacher: document.getElementById('filter-teacher'),
    filterRoom: document.getElementById('filter-room'),
    filterType: document.getElementById('filter-type'),
    filterSearch: document.getElementById('filter-search'),
    btnClearFilters: document.getElementById('btn-clear-filters'),
    btnViewGrid: document.getElementById('btn-view-grid'),
    btnViewList: document.getElementById('btn-view-list'),
    activitiesCount: document.getElementById('activities-count'),
    gridView: document.getElementById('grid-view'),
    listView: document.getElementById('list-view'),
    scheduleGrid: document.getElementById('schedule-grid'),
    activitiesTbody: document.getElementById('activities-tbody'),
    statTotal: document.getElementById('stat-total'),
    statCourses: document.getElementById('stat-courses'),
    statRooms: document.getElementById('stat-rooms'),
    statGenerated: document.getElementById('stat-generated')
};

async function init() {
    try {
        await loadScheduleData();
        populateFilters();
        setupEventListeners();
        applyFilters();
        renderCurrentView();
        updateStats();
    } catch (error) {
        console.error('Error initializing app:', error);
        showError('Error al cargar los datos del horario');
    }
}

async function loadScheduleData() {
    const response = await fetch('/data/output/schedule.json');
    if (!response.ok) {
        throw new Error('Failed to load schedule.json');
    }
    state.scheduleData = await response.json();

    state.allActivities = [];
    for (const day of state.scheduleData.schedule) {
        for (const block of day.blocks) {
            for (const activity of block.activities) {
                state.allActivities.push({
                    ...activity,
                    dayName: day.day
                });
            }
        }
    }

    state.filteredActivities = [...state.allActivities];
}

function populateFilters() {
    DAYS.forEach(day => {
        const option = document.createElement('option');
        option.value = day;
        option.textContent = day;
        elements.filterDay.appendChild(option);
    });

    const courses = [...new Set(state.allActivities.map(a => a.course_code))].sort();
    courses.forEach(course => {
        const option = document.createElement('option');
        option.value = course;
        option.textContent = course;
        elements.filterCourse.appendChild(option);
    });

    const teachers = [...new Set(state.allActivities.flatMap(a => a.teachers || []))]
        .filter(t => t && t.trim())
        .sort();
    teachers.forEach(teacher => {
        const option = document.createElement('option');
        option.value = teacher;
        option.textContent = teacher;
        elements.filterTeacher.appendChild(option);
    });

    const rooms = [...new Set(state.allActivities.map(a => a.room))].sort();
    rooms.forEach(room => {
        const option = document.createElement('option');
        option.value = room;
        option.textContent = room;
        elements.filterRoom.appendChild(option);
    });
}

function setupEventListeners() {
    elements.filterDay.addEventListener('change', () => {
        state.filters.day = elements.filterDay.value;
        applyFilters();
        renderCurrentView();
    });

    elements.filterCourse.addEventListener('change', () => {
        state.filters.course = elements.filterCourse.value;
        applyFilters();
        renderCurrentView();
    });

    elements.filterTeacher.addEventListener('change', () => {
        state.filters.teacher = elements.filterTeacher.value;
        applyFilters();
        renderCurrentView();
    });

    elements.filterRoom.addEventListener('change', () => {
        state.filters.room = elements.filterRoom.value;
        applyFilters();
        renderCurrentView();
    });

    elements.filterType.addEventListener('change', () => {
        state.filters.type = elements.filterType.value;
        applyFilters();
        renderCurrentView();
    });

    elements.filterSearch.addEventListener('input', () => {
        state.filters.search = elements.filterSearch.value.toLowerCase();
        applyFilters();
        renderCurrentView();
    });

    elements.btnClearFilters.addEventListener('click', clearFilters);

    elements.btnViewGrid.addEventListener('click', () => setView('grid'));
    elements.btnViewList.addEventListener('click', () => setView('list'));
}

function applyFilters() {
    state.filteredActivities = state.allActivities.filter(activity => {
        if (state.filters.day && activity.dayName !== state.filters.day) {
            return false;
        }

        if (state.filters.course && activity.course_code !== state.filters.course) {
            return false;
        }

        if (state.filters.teacher) {
            const teachers = activity.teachers || [];
            if (!teachers.includes(state.filters.teacher)) {
                return false;
            }
        }

        if (state.filters.room && activity.room !== state.filters.room) {
            return false;
        }

        if (state.filters.type && activity.type !== state.filters.type) {
            return false;
        }

        if (state.filters.search) {
            const searchText = [
                activity.code,
                activity.course_code,
                activity.course_name,
                activity.room,
                ...(activity.teachers || [])
            ].join(' ').toLowerCase();

            if (!searchText.includes(state.filters.search)) {
                return false;
            }
        }

        return true;
    });

    elements.activitiesCount.textContent = `${state.filteredActivities.length} actividades`;
}

function clearFilters() {
    state.filters = { day: '', course: '', teacher: '', room: '', type: '', search: '' };

    elements.filterDay.value = '';
    elements.filterCourse.value = '';
    elements.filterTeacher.value = '';
    elements.filterRoom.value = '';
    elements.filterType.value = '';
    elements.filterSearch.value = '';

    applyFilters();
    renderCurrentView();
}

function setView(view) {
    state.view = view;

    elements.btnViewGrid.classList.toggle('active', view === 'grid');
    elements.btnViewList.classList.toggle('active', view === 'list');

    elements.gridView.classList.toggle('hidden', view !== 'grid');
    elements.listView.classList.toggle('hidden', view !== 'list');

    renderCurrentView();
}

function renderCurrentView() {
    if (state.view === 'grid') {
        renderGridView();
    } else {
        renderListView();
    }
}

function renderGridView() {
    const grid = elements.scheduleGrid;
    grid.innerHTML = '';

    grid.appendChild(createGridHeader('Hora'));
    DAYS.forEach(day => grid.appendChild(createGridHeader(day)));

    TIME_SLOTS.forEach((slot, slotIndex) => {
        const timeCell = document.createElement('div');
        timeCell.className = 'grid-time';
        timeCell.innerHTML = `
            <span class="block-num">Bloque ${slotIndex}</span>
            <span>${slot.time}</span>
        `;
        grid.appendChild(timeCell);

        DAYS.forEach((day, dayIndex) => {
            const blockNum = dayIndex * BLOCKS_PER_DAY + slotIndex;
            const cell = document.createElement('div');
            cell.className = 'grid-cell';

            if (blockNum === PROTECTED_BLOCK) {
                cell.classList.add('protected');
            } else {
                const cellActivities = state.filteredActivities.filter(a =>
                    a.dayName === day && a.block === blockNum
                );

                cellActivities.forEach(activity => {
                    cell.appendChild(createActivityCard(activity));
                });
            }

            grid.appendChild(cell);
        });
    });
}

function createGridHeader(text) {
    const header = document.createElement('div');
    header.className = 'grid-header';
    header.textContent = text;
    return header;
}

function createActivityCard(activity) {
    const card = document.createElement('div');
    const typeClass = activity.type.toLowerCase().replace('í', 'i');
    card.className = `activity-card type-${typeClass}`;

    card.innerHTML = `
        <div class="code">${activity.code}</div>
        <div class="room">${activity.room}</div>
    `;

    card.addEventListener('click', () => showActivityModal(activity));

    return card;
}
function renderListView() {
    const tbody = elements.activitiesTbody;
    tbody.innerHTML = '';

    const sorted = [...state.filteredActivities].sort((a, b) => {
        const dayDiff = DAYS.indexOf(a.dayName) - DAYS.indexOf(b.dayName);
        if (dayDiff !== 0) return dayDiff;
        return a.block - b.block;
    });

    sorted.forEach(activity => {
        const row = document.createElement('tr');
        const typeClass = activity.type.toLowerCase().replace('í', 'i');
        const typeLabel = activity.type === 'CATEDRA' ? 'CAT' :
            activity.type === 'AYUDANTIA' ? 'AYU' : 'LAB';

        row.innerHTML = `
            <td><strong>${activity.code}</strong></td>
            <td>${activity.course_name}</td>
            <td><span class="type-badge ${typeClass}">${typeLabel}</span></td>
            <td>${activity.dayName}</td>
            <td>${activity.time_slot}</td>
            <td>${activity.room}</td>
            <td>${(activity.teachers || []).join(', ') || '-'}</td>
            <td>${(activity.sections || []).join(', ')}</td>
            <td>${activity.students}</td>
        `;

        row.addEventListener('click', () => showActivityModal(activity));
        tbody.appendChild(row);
    });
}

function showActivityModal(activity) {
    const existingModal = document.querySelector('.modal-overlay');
    if (existingModal) existingModal.remove();

    const modal = document.createElement('div');
    modal.className = 'modal-overlay';

    modal.innerHTML = `
        <div class="modal-content">
            <h3>${activity.code}</h3>
            <div class="detail-row">
                <span class="detail-label">Curso</span>
                <span>${activity.course_name}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">Código</span>
                <span>${activity.course_code}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">Tipo</span>
                <span>${activity.type}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">Día</span>
                <span>${activity.dayName}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">Hora</span>
                <span>${activity.time_slot}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">Sala</span>
                <span>${activity.room}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">Profesor(es)</span>
                <span>${(activity.teachers || []).join(', ') || 'Sin asignar'}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">Secciones</span>
                <span>${(activity.sections || []).join(', ')}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">Estudiantes</span>
                <span>${activity.students}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">Duración</span>
                <span>${activity.duration} bloque(s)</span>
            </div>
            <button class="btn-close">Cerrar</button>
        </div>
    `;

    modal.addEventListener('click', (e) => {
        if (e.target === modal || e.target.classList.contains('btn-close')) {
            modal.remove();
        }
    });

    document.body.appendChild(modal);
}

function updateStats() {
    const data = state.scheduleData;

    elements.statTotal.textContent = data.summary.total_activities;
    elements.statCourses.textContent = data.summary.total_courses;
    elements.statRooms.textContent = data.summary.total_rooms;
    elements.statGenerated.textContent = data.generated_at.split(' ')[0]; // Just date
}

function showError(message) {
    elements.scheduleGrid.innerHTML = `
        <div style="grid-column: 1/-1; padding: 2rem; text-align: center; color: #ef4444;">
            <h3>⚠️ Error</h3>
            <p>${message}</p>
            <p style="font-size: 0.875rem; color: #64748b;">
                Asegúrate de que el archivo <code>data/output/schedule.json</code> existe.
            </p>
        </div>
    `;
}

document.addEventListener('DOMContentLoaded', init);
