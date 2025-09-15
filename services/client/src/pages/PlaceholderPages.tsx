import React, { useEffect, useState, useMemo } from 'react';
import { Plus, Filter, Search, Edit, Trash2, Tag, Calendar, CheckCircle, XCircle, Clock, AlertTriangle, MoreVertical, Play, Pause, FolderOpen, ArrowLeft, MessageSquare } from 'lucide-react';
import { useTasksStore, useNotesStore, useProjectsStore } from '../store';
import { tasksAPI, TaskResponse, TaskStatus, TaskPriority, CreateTaskRequest, UpdateTaskRequest, UpdateTaskStatusRequest, notesAPI, projectsAPI, NoteResponse, ProjectResponse, ProjectStatus, ProjectPriority } from '../services/api';

export const NotesPage: React.FC = () => {
    const {
        notes,
        isLoading,
        error,
        setNotes,
        setLoading,
        setError,
        addNote,
        updateNote,
        removeNote
    } = useNotesStore();

    const [searchQuery, setSearchQuery] = useState('');
    const [selectedTags, setSelectedTags] = useState<string[]>([]);
    const [showCreateForm, setShowCreateForm] = useState(false);
    const [editingNote, setEditingNote] = useState<NoteResponse | null>(null);

    useEffect(() => {
        loadNotes();
    }, []);

    const loadNotes = async () => {
        try {
            setLoading(true);
            setError(null);
            const response = await notesAPI.listNotes({
                search: searchQuery || undefined,
                tags: selectedTags.length > 0 ? selectedTags : undefined,
                orderBy: 'created_at',
                order: 'desc',
                limit: 50
            });
            setNotes(response.notes, response.pagination);
        } catch (err: any) {
            setError(err.message || 'Failed to load notes');
        } finally {
            setLoading(false);
        }
    };

    const handleSearch = async () => {
        await loadNotes();
    };

    const handleCreateNote = async (noteData: { content: string; tags: string[]; projectId?: string }) => {
        try {
            setLoading(true);
            const newNote = await notesAPI.createNote(noteData);
            addNote(newNote);
            setShowCreateForm(false);
        } catch (err: any) {
            setError(err.message || 'Failed to create note');
        } finally {
            setLoading(false);
        }
    };

    const handleUpdateNote = async (noteId: string, updates: { content?: string; tags?: string[]; projectId?: string }) => {
        try {
            setLoading(true);
            const updatedNote = await notesAPI.updateNote(noteId, updates);
            updateNote(noteId, updatedNote);
            setEditingNote(null);
        } catch (err: any) {
            setError(err.message || 'Failed to update note');
        } finally {
            setLoading(false);
        }
    };

    const handleDeleteNote = async (noteId: string) => {
        if (!confirm('Are you sure you want to delete this note?')) return;

        try {
            setLoading(true);
            await notesAPI.deleteNote(noteId);
            removeNote(noteId);
        } catch (err: any) {
            setError(err.message || 'Failed to delete note');
        } finally {
            setLoading(false);
        }
    };

    const formatDate = (dateString: string) => {
        return new Date(dateString).toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    };

    return (
        <div className="page-container">
            <div className="page-header">
                <div className="header-left">
                    <h1>Notes</h1>
                    <p>Capture and organize your thoughts over time</p>
                </div>
                <div className="header-actions">
                    <div className="search-bar">
                        <input
                            type="text"
                            placeholder="Search notes..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
                        />
                        <button onClick={handleSearch}>
                            <Search size={16} />
                        </button>
                    </div>
                    <button className="action-button secondary">
                        <Filter size={16} />
                        Filter
                    </button>
                    <button
                        className="action-button"
                        onClick={() => setShowCreateForm(true)}
                    >
                        <Plus size={16} />
                        New Note
                    </button>
                </div>
            </div>

            <div className="page-content">
                {error && (
                    <div className="error-message">
                        {error}
                    </div>
                )}

                {isLoading && notes.length === 0 ? (
                    <div className="loading-state">
                        <div className="spinner"></div>
                        <p>Loading notes...</p>
                    </div>
                ) : notes.length === 0 ? (
                    <div className="empty-state">
                        <div className="empty-icon">📝</div>
                        <h3>No Notes Yet</h3>
                        <p>Start capturing your thoughts and ideas. Your notes will be stored and organized over time.</p>
                        <button
                            className="cta-button"
                            onClick={() => setShowCreateForm(true)}
                        >
                            <Plus size={16} />
                            Create Note
                        </button>
                    </div>
                ) : (
                    <div className="notes-grid">
                        {notes.map((note) => (
                            <div key={note.id} className="note-card">
                                <div className="note-header">
                                    <div className="note-date">
                                        <Calendar size={14} />
                                        {formatDate(note.createdAt)}
                                    </div>
                                    <div className="note-actions">
                                        <button
                                            onClick={() => setEditingNote(note)}
                                            className="action-button small"
                                        >
                                            <Edit size={14} />
                                        </button>
                                        <button
                                            onClick={() => handleDeleteNote(note.id)}
                                            className="action-button small danger"
                                        >
                                            <Trash2 size={14} />
                                        </button>
                                    </div>
                                </div>

                                <div className="note-content">
                                    {note.content.length > 200
                                        ? `${note.content.substring(0, 200)}...`
                                        : note.content
                                    }
                                </div>

                                {note.tags && note.tags.length > 0 && (
                                    <div className="note-tags">
                                        {note.tags.map((tag, index) => (
                                            <span key={index} className="tag">
                                                <Tag size={12} />
                                                {tag}
                                            </span>
                                        ))}
                                    </div>
                                )}

                                {note.projectId && (
                                    <div className="note-project">
                                        <span className="project-badge">
                                            Project: {note.projectId}
                                        </span>
                                    </div>
                                )}
                            </div>
                        ))}
                    </div>
                )}

                {showCreateForm && (
                    <NoteForm
                        onSubmit={handleCreateNote}
                        onCancel={() => setShowCreateForm(false)}
                        isLoading={isLoading}
                    />
                )}

                {editingNote && (
                    <NoteForm
                        note={editingNote}
                        onSubmit={(data) => handleUpdateNote(editingNote.id, data)}
                        onCancel={() => setEditingNote(null)}
                        isLoading={isLoading}
                    />
                )}
            </div>
        </div>
    );
};

interface NoteFormProps {
    note?: NoteResponse;
    onSubmit: (data: { content: string; tags: string[]; projectId?: string }) => void;
    onCancel: () => void;
    isLoading: boolean;
}

const NoteForm: React.FC<NoteFormProps> = ({ note, onSubmit, onCancel, isLoading }) => {
    const [content, setContent] = useState(note?.content || '');
    const [tags, setTags] = useState(note?.tags?.join(', ') || '');
    const [projectId, setProjectId] = useState(note?.projectId || '');

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        const tagArray = tags.split(',').map(t => t.trim()).filter(t => t);
        onSubmit({
            content,
            tags: tagArray,
            projectId: projectId || undefined
        });
    };

    return (
        <div className="modal-overlay">
            <div className="modal">
                <div className="modal-header">
                    <h3>{note ? 'Edit Note' : 'Create New Note'}</h3>
                </div>
                <form onSubmit={handleSubmit} className="modal-body">
                    <div className="form-group">
                        <label htmlFor="content">Content</label>
                        <textarea
                            id="content"
                            value={content}
                            onChange={(e) => setContent(e.target.value)}
                            placeholder="Write your note here..."
                            rows={6}
                            required
                        />
                    </div>

                    <div className="form-group">
                        <label htmlFor="tags">Tags (comma-separated)</label>
                        <input
                            id="tags"
                            type="text"
                            value={tags}
                            onChange={(e) => setTags(e.target.value)}
                            placeholder="work, ideas, personal"
                        />
                    </div>

                    <div className="form-group">
                        <label htmlFor="projectId">Project ID (optional)</label>
                        <input
                            id="projectId"
                            type="text"
                            value={projectId}
                            onChange={(e) => setProjectId(e.target.value)}
                            placeholder="Link to a project"
                        />
                    </div>

                    <div className="modal-actions">
                        <button type="button" onClick={onCancel} className="cancel-button">
                            Cancel
                        </button>
                        <button type="submit" disabled={isLoading || !content.trim()}>
                            {isLoading ? 'Saving...' : (note ? 'Update' : 'Create')}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export const ProjectsPage: React.FC = () => {
    const {
        projects,
        currentProject,
        isLoading,
        error,
        setProjects,
        setCurrentProject,
        setLoading,
        setError,
        addProject,
        updateProject,
        removeProject
    } = useProjectsStore();

    const [showCreateForm, setShowCreateForm] = useState(false);
    const [editingProject, setEditingProject] = useState<ProjectResponse | null>(null);
    const [filterStatus, setFilterStatus] = useState<string>('');

    useEffect(() => {
        loadProjects();
    }, [filterStatus]);

    const loadProjects = async () => {
        try {
            setLoading(true);
            setError(null);
            const response = await projectsAPI.listProjects({
                status: filterStatus as any || undefined,
                limit: 50
            });
            setProjects(response.projects, response.pagination);
        } catch (err: any) {
            setError(err.message || 'Failed to load projects');
        } finally {
            setLoading(false);
        }
    };

    const handleCreateProject = async (projectData: {
        name: string;
        description?: string;
        status?: ProjectStatus;
        priority?: ProjectPriority;
        tags?: string[];
        dueAt?: string;
    }) => {
        try {
            setLoading(true);
            const newProject = await projectsAPI.createProject(projectData);
            addProject(newProject);
            setShowCreateForm(false);
        } catch (err: any) {
            setError(err.message || 'Failed to create project');
        } finally {
            setLoading(false);
        }
    };

    const handleUpdateProject = async (projectId: string, updates: any) => {
        try {
            setLoading(true);
            const updatedProject = await projectsAPI.updateProject(projectId, updates);
            updateProject(projectId, updatedProject);
            setEditingProject(null);
        } catch (err: any) {
            setError(err.message || 'Failed to update project');
        } finally {
            setLoading(false);
        }
    };

    const handleDeleteProject = async (projectId: string) => {
        if (!confirm('Are you sure you want to delete this project?')) return;

        try {
            setLoading(true);
            await projectsAPI.deleteProject(projectId);
            removeProject(projectId);
        } catch (err: any) {
            setError(err.message || 'Failed to delete project');
        } finally {
            setLoading(false);
        }
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'planned': return '#6b7280';
            case 'in_progress': return '#3b82f6';
            case 'blocked': return '#ef4444';
            case 'done': return '#10b981';
            case 'archived': return '#9ca3af';
            default: return '#6b7280';
        }
    };

    const getPriorityColor = (priority: string) => {
        switch (priority) {
            case 'urgent': return '#ef4444';
            case 'high': return '#f59e0b';
            case 'med': return '#3b82f6';
            case 'low': return '#6b7280';
            default: return '#6b7280';
        }
    };

    if (currentProject) {
        return (
            <ProjectDetailView
                project={currentProject}
                onBack={() => setCurrentProject(null)}
                onUpdate={(updates) => handleUpdateProject(currentProject.id, updates)}
                onDelete={() => handleDeleteProject(currentProject.id)}
                isLoading={isLoading}
            />
        );
    }

    return (
        <div className="page-container">
            <div className="page-header">
                <div className="header-left">
                    <h1>Projects</h1>
                    <p>Organize and manage your AI-assisted projects</p>
                </div>
                <div className="header-actions">
                    <select
                        value={filterStatus}
                        onChange={(e) => setFilterStatus(e.target.value)}
                        className="filter-select"
                    >
                        <option value="">All Status</option>
                        <option value="planned">Planned</option>
                        <option value="in_progress">In Progress</option>
                        <option value="blocked">Blocked</option>
                        <option value="done">Done</option>
                        <option value="archived">Archived</option>
                    </select>
                    <button
                        className="action-button"
                        onClick={() => setShowCreateForm(true)}
                    >
                        <Plus size={16} />
                        New Project
                    </button>
                </div>
            </div>

            <div className="page-content">
                {error && (
                    <div className="error-message">
                        {error}
                    </div>
                )}

                {isLoading && projects.length === 0 ? (
                    <div className="loading-state">
                        <div className="spinner"></div>
                        <p>Loading projects...</p>
                    </div>
                ) : projects.length === 0 ? (
                    <div className="empty-state">
                        <FolderOpen size={48} opacity={0.5} />
                        <h3>No Projects Yet</h3>
                        <p>Create your first project to get started with organizing your work.</p>
                        <button
                            className="cta-button"
                            onClick={() => setShowCreateForm(true)}
                        >
                            <Plus size={16} />
                            Create Project
                        </button>
                    </div>
                ) : (
                    <div className="projects-grid">
                        {projects.map((project) => (
                            <div
                                key={project.id}
                                className="project-card"
                                onClick={() => setCurrentProject(project)}
                            >
                                <div className="project-header">
                                    <h3>{project.name}</h3>
                                    <div className="project-badges">
                                        <span
                                            className="status-badge"
                                            style={{ backgroundColor: getStatusColor(project.status) }}
                                        >
                                            {project.status.replace('_', ' ')}
                                        </span>
                                        <span
                                            className="priority-badge"
                                            style={{ backgroundColor: getPriorityColor(project.priority) }}
                                        >
                                            {project.priority}
                                        </span>
                                    </div>
                                </div>

                                {project.description && (
                                    <p className="project-description">
                                        {project.description.length > 150
                                            ? `${project.description.substring(0, 150)}...`
                                            : project.description
                                        }
                                    </p>
                                )}

                                <div className="project-meta">
                                    {project.dueAt && (
                                        <div className="due-date">
                                            <Calendar size={14} />
                                            Due: {new Date(project.dueAt).toLocaleDateString()}
                                        </div>
                                    )}
                                    <div className="progress-count">
                                        {project.progress?.length || 0} updates
                                    </div>
                                </div>

                                {project.tags && project.tags.length > 0 && (
                                    <div className="project-tags">
                                        {project.tags.slice(0, 3).map((tag, index) => (
                                            <span key={index} className="tag">
                                                <Tag size={12} />
                                                {tag}
                                            </span>
                                        ))}
                                        {project.tags.length > 3 && (
                                            <span className="tag more">+{project.tags.length - 3}</span>
                                        )}
                                    </div>
                                )}
                            </div>
                        ))}
                    </div>
                )}

                {showCreateForm && (
                    <ProjectForm
                        onSubmit={handleCreateProject}
                        onCancel={() => setShowCreateForm(false)}
                        isLoading={isLoading}
                    />
                )}

                {editingProject && (
                    <ProjectForm
                        project={editingProject}
                        onSubmit={(data) => handleUpdateProject(editingProject.id, data)}
                        onCancel={() => setEditingProject(null)}
                        isLoading={isLoading}
                    />
                )}
            </div>
        </div>
    );
};

interface ProjectDetailViewProps {
    project: ProjectResponse;
    onBack: () => void;
    onUpdate: (updates: any) => void;
    onDelete: () => void;
    isLoading: boolean;
}

const ProjectDetailView: React.FC<ProjectDetailViewProps> = ({
    project,
    onBack,
    onUpdate,
    onDelete,
    isLoading
}) => {
    const [projectNotes, setProjectNotes] = useState<NoteResponse[]>([]);

    useEffect(() => {
        loadProjectNotes();
    }, []);

    const loadProjectNotes = async () => {
        try {
            const res = await notesAPI.listNotes({ orderBy: 'created_at', order: 'desc', limit: 200 });
            const filtered = (res.notes || []).filter((n) => n.projectId === project.id);
            setProjectNotes(filtered);
        } catch (err) {
            console.error('Failed to load project notes:', err);
        }
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'planned': return '#6b7280';
            case 'in_progress': return '#3b82f6';
            case 'blocked': return '#ef4444';
            case 'done': return '#10b981';
            case 'archived': return '#9ca3af';
            default: return '#6b7280';
        }
    };

    const getPriorityColor = (priority: string) => {
        switch (priority) {
            case 'urgent': return '#ef4444';
            case 'high': return '#f59e0b';
            case 'med': return '#3b82f6';
            case 'low': return '#6b7280';
            default: return '#6b7280';
        }
    };

    // Combined timeline: created/meta, progress events, notes
    type TimelineItem = {
        id: string;
        at: string;
        kind: 'created' | 'note' | 'progress';
        label: string;
        details?: string;
        tags?: string[];
        by?: string;
    };

    const timeline: TimelineItem[] = useMemo(() => {
        const items: TimelineItem[] = [];

        items.push({
            id: `created-${project.id}`,
            at: project.createdAt,
            kind: 'created',
            label: 'Project created',
            details: project.description || undefined,
            tags: project.tags || [],
        });

        if (project.progress && project.progress.length > 0) {
            project.progress.forEach((ev, idx) => {
                items.push({
                    id: `progress-${idx}-${ev.at}`,
                    at: ev.at,
                    kind: 'progress',
                    label: ev.kind,
                    details: ev.memo,
                    by: ev.by,
                });
            });
        }

        if (projectNotes && projectNotes.length > 0) {
            projectNotes.forEach((n) => {
                items.push({
                    id: `note-${n.id}`,
                    at: n.createdAt,
                    kind: 'note',
                    label: 'Note added',
                    details: n.content,
                    tags: n.tags,
                });
            });
        }

        items.sort((a, b) => new Date(b.at).getTime() - new Date(a.at).getTime());
        return items;
    }, [project, projectNotes]);

    const [showAllTimeline, setShowAllTimeline] = useState(false);
    const visibleTimeline = useMemo(() => {
        const MAX = 50;
        return showAllTimeline ? timeline : timeline.slice(0, MAX);
    }, [timeline, showAllTimeline]);

    return (
        <div className="project-detail">
            <div className="project-detail-header">
                <button onClick={onBack} className="back-button">
                    <ArrowLeft size={16} />
                    Back to Projects
                </button>
                <div className="project-title-section">
                    <h1>{project.name}</h1>
                </div>
            </div>

            <div className="project-content">
                <div className="notebook-content journal-page">
                    <div className="project-meta-inline">
                        <span className="meta-pill" style={{ backgroundColor: getStatusColor(project.status) }}>
                            {project.status.replace('_', ' ')}
                        </span>
                        <span className="meta-pill" style={{ backgroundColor: getPriorityColor(project.priority) }}>
                            {project.priority}
                        </span>
                        {project.dueAt && (
                            <span className="meta-text"><Calendar size={14} /> Due {new Date(project.dueAt).toLocaleDateString()}</span>
                        )}
                        <span className="meta-text"><Clock size={14} /> Created {new Date(project.createdAt).toLocaleDateString()}</span>
                    </div>

                    {project.tags && project.tags.length > 0 && (
                        <div className="tags-list" style={{ marginTop: '0.75rem' }}>
                            {project.tags.map((tag, index) => (
                                <span key={index} className="tag"><Tag size={12} />{tag}</span>
                            ))}
                        </div>
                    )}

                    <div className="article-body">
                        {project.description || 'No description provided.'}
                    </div>

                    {/* Journal feed */}
                    <div className="notebook-section">
                        {timeline.length === 0 ? (
                            <p style={{ color: '#888', fontStyle: 'italic' }}>No entries yet.</p>
                        ) : (
                            <div className="timeline">
                                {visibleTimeline.map((item) => (
                                    <div key={item.id} className={`timeline-item ${item.kind}`}>
                                        <div className="timeline-time">
                                            <Clock size={14} /> {new Date(item.at).toLocaleString()}
                                        </div>
                                        <div className="timeline-content">
                                            <div className="timeline-label">
                                                {item.kind === 'created' && (<><FolderOpen size={16} /> Project created</>)}
                                                {item.kind === 'note' && (<><MessageSquare size={16} /> Note</>)}
                                                {item.kind === 'progress' && (<><Calendar size={16} /> {item.label}</>)}
                                                {item.by && <span className="timeline-by"> by {item.by}</span>}
                                            </div>
                                            {item.details && (
                                                <div className="timeline-details">
                                                    {item.details.length > 500 ? `${item.details.slice(0, 500)}…` : item.details}
                                                </div>
                                            )}
                                            {item.tags && item.tags.length > 0 && (
                                                <div className="timeline-tags">
                                                    {item.tags.map((t, i) => (
                                                        <span key={i} className="tag"><Tag size={12} />{t}</span>
                                                    ))}
                                                </div>
                                            )}
                                        </div>
                                    </div>
                                ))}

                                {timeline.length > visibleTimeline.length && (
                                    <div style={{ display: 'flex', justifyContent: 'center', marginTop: '1rem' }}>
                                        <button className="action-button" onClick={() => setShowAllTimeline(true)}>
                                            Show older entries
                                        </button>
                                    </div>
                                )}
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
};

interface ProjectFormProps {
    project?: ProjectResponse;
    onSubmit: (data: any) => void;
    onCancel: () => void;
    isLoading: boolean;
}

const ProjectForm: React.FC<ProjectFormProps> = ({ project, onSubmit, onCancel, isLoading }) => {
    const [name, setName] = useState(project?.name || '');
    const [description, setDescription] = useState(project?.description || '');
    const [status, setStatus] = useState(project?.status || 'planned');
    const [priority, setPriority] = useState(project?.priority || 'med');
    const [tags, setTags] = useState(project?.tags?.join(', ') || '');
    const [dueAt, setDueAt] = useState(project?.dueAt ? new Date(project.dueAt).toISOString().split('T')[0] : '');

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        const tagArray = tags.split(',').map(t => t.trim()).filter(t => t);
        onSubmit({
            name,
            description: description || undefined,
            status: status as ProjectStatus,
            priority: priority as ProjectPriority,
            tags: tagArray,
            dueAt: dueAt || undefined
        });
    };

    return (
        <div className="modal-overlay">
            <div className="modal">
                <div className="modal-header">
                    <h3>{project ? 'Edit Project' : 'Create New Project'}</h3>
                </div>
                <form onSubmit={handleSubmit} className="modal-body">
                    <div className="form-group">
                        <label htmlFor="name">Name *</label>
                        <input
                            id="name"
                            type="text"
                            value={name}
                            onChange={(e) => setName(e.target.value)}
                            placeholder="Project name"
                            required
                        />
                    </div>

                    <div className="form-group">
                        <label htmlFor="description">Description</label>
                        <textarea
                            id="description"
                            value={description}
                            onChange={(e) => setDescription(e.target.value)}
                            placeholder="Project description"
                            rows={3}
                        />
                    </div>

                    <div className="form-row">
                        <div className="form-group">
                            <label htmlFor="status">Status</label>
                            <select
                                id="status"
                                value={status}
                                onChange={(e) => setStatus(e.target.value as ProjectStatus)}
                            >
                                <option value="planned">Planned</option>
                                <option value="in_progress">In Progress</option>
                                <option value="blocked">Blocked</option>
                                <option value="done">Done</option>
                                <option value="archived">Archived</option>
                            </select>
                        </div>

                        <div className="form-group">
                            <label htmlFor="priority">Priority</label>
                            <select
                                id="priority"
                                value={priority}
                                onChange={(e) => setPriority(e.target.value as ProjectPriority)}
                            >
                                <option value="low">Low</option>
                                <option value="med">Medium</option>
                                <option value="high">High</option>
                                <option value="urgent">Urgent</option>
                            </select>
                        </div>
                    </div>

                    <div className="form-group">
                        <label htmlFor="dueAt">Due Date</label>
                        <input
                            id="dueAt"
                            type="date"
                            value={dueAt}
                            onChange={(e) => setDueAt(e.target.value)}
                        />
                    </div>

                    <div className="form-group">
                        <label htmlFor="tags">Tags (comma-separated)</label>
                        <input
                            id="tags"
                            type="text"
                            value={tags}
                            onChange={(e) => setTags(e.target.value)}
                            placeholder="web, design, urgent"
                        />
                    </div>

                    <div className="modal-actions">
                        <button type="button" onClick={onCancel} className="cancel-button">
                            Cancel
                        </button>
                        <button type="submit" disabled={isLoading || !name.trim()}>
                            {isLoading ? 'Saving...' : (project ? 'Update' : 'Create')}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export const TasksPage: React.FC = () => {
    const {
        tasks,
        isLoading,
        error,
        setTasks,
        setLoading,
        setError,
        addTask,
        updateTask,
        removeTask
    } = useTasksStore();

    const [searchQuery, setSearchQuery] = useState('');
    const [selectedStatus, setSelectedStatus] = useState<TaskStatus | ''>('');
    const [selectedPriority, setSelectedPriority] = useState<TaskPriority | ''>('');
    const [selectedTags, setSelectedTags] = useState<string[]>([]);
    const [sortBy, setSortBy] = useState<'createdAt' | 'dueAt' | 'priority' | 'title'>('createdAt');
    const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');
    const [showCreateForm, setShowCreateForm] = useState(false);
    const [editingTask, setEditingTask] = useState<TaskResponse | null>(null);

    useEffect(() => {
        loadTasks();
    }, [selectedStatus, selectedPriority, selectedTags, sortBy, sortOrder]);

    const loadTasks = async () => {
        try {
            setLoading(true);
            setError(null);
            const response = await tasksAPI.listTasks({
                search: searchQuery || undefined,
                status: selectedStatus as TaskStatus || undefined,
                priority: selectedPriority as TaskPriority || undefined,
                tags: selectedTags.length > 0 ? selectedTags : undefined,
                orderBy: sortBy,
                order: sortOrder,
                limit: 100
            });
            setTasks(response.tasks, response.pagination);
        } catch (err: any) {
            setError(err.message || 'Failed to load tasks');
        } finally {
            setLoading(false);
        }
    };

    const handleSearch = async () => {
        await loadTasks();
    };

    const handleCreateTask = async (taskData: CreateTaskRequest) => {
        try {
            setLoading(true);
            const newTask = await tasksAPI.createTask(taskData);
            addTask(newTask);
            setShowCreateForm(false);
        } catch (err: any) {
            setError(err.message || 'Failed to create task');
        } finally {
            setLoading(false);
        }
    };

    const handleUpdateTask = async (taskId: string, updates: UpdateTaskRequest) => {
        try {
            setLoading(true);
            const updatedTask = await tasksAPI.updateTask(taskId, updates);
            updateTask(taskId, updatedTask);
            setEditingTask(null);
        } catch (err: any) {
            setError(err.message || 'Failed to update task');
        } finally {
            setLoading(false);
        }
    };

    const handleDeleteTask = async (taskId: string) => {
        if (!confirm('Are you sure you want to delete this task?')) return;

        try {
            setLoading(true);
            await tasksAPI.deleteTask(taskId);
            removeTask(taskId);
        } catch (err: any) {
            setError(err.message || 'Failed to delete task');
        } finally {
            setLoading(false);
        }
    };

    const handleStatusChange = async (taskId: string, status: TaskStatus) => {
        try {
            setLoading(true);
            const updatedTask = await tasksAPI.updateTaskStatus(taskId, { status });
            updateTask(taskId, updatedTask);
        } catch (err: any) {
            setError(err.message || 'Failed to update task status');
        } finally {
            setLoading(false);
        }
    };

    const getStatusIcon = (status: TaskStatus) => {
        switch (status) {
            case 'done':
                return <CheckCircle size={16} className="status-icon done" />;
            case 'cancelled':
                return <XCircle size={16} className="status-icon cancelled" />;
            case 'pending':
            default:
                return <Clock size={16} className="status-icon pending" />;
        }
    };

    const getStatusColor = (status: TaskStatus) => {
        switch (status) {
            case 'done': return '#10b981';
            case 'cancelled': return '#ef4444';
            case 'pending': return '#f59e0b';
            default: return '#6b7280';
        }
    };

    const getPriorityColor = (priority: TaskPriority) => {
        switch (priority) {
            case 5: return '#ef4444';
            case 4: return '#f97316';
            case 3: return '#f59e0b';
            case 2: return '#3b82f6';
            case 1: return '#6b7280';
            default: return '#6b7280';
        }
    };

    const getPriorityLabel = (priority: TaskPriority) => {
        switch (priority) {
            case 5: return 'Urgent';
            case 4: return 'High';
            case 3: return 'Medium';
            case 2: return 'Low';
            case 1: return 'Lowest';
            default: return 'Unknown';
        }
    };

    const formatDate = (dateString?: string) => {
        if (!dateString) return 'No date';
        return new Date(dateString).toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    };

    const isOverdue = (dueAt?: string) => {
        if (!dueAt) return false;
        return new Date(dueAt) < new Date() && !dueAt.includes('completed') && !dueAt.includes('cancelled');
    };

    return (
        <div className="page-container">
            <div className="page-header">
                <div className="header-left">
                    <h1>Tasks</h1>
                    <p>Track and manage your tasks with AI assistance</p>
                </div>
                <div className="header-actions">
                    <div className="search-bar">
                        <input
                            type="text"
                            placeholder="Search tasks..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
                        />
                        <button onClick={handleSearch}>
                            <Search size={16} />
                        </button>
                    </div>
                    <select
                        value={selectedStatus}
                        onChange={(e) => setSelectedStatus(e.target.value as TaskStatus | '')}
                        className="filter-select"
                    >
                        <option value="">All Status</option>
                        <option value="pending">Pending</option>
                        <option value="done">Done</option>
                        <option value="cancelled">Cancelled</option>
                    </select>
                    <select
                        value={selectedPriority}
                        onChange={(e) => setSelectedPriority(e.target.value ? parseInt(e.target.value) as TaskPriority : '')}
                        className="filter-select"
                    >
                        <option value="">All Priority</option>
                        <option value="5">Urgent</option>
                        <option value="4">High</option>
                        <option value="3">Medium</option>
                        <option value="2">Low</option>
                        <option value="1">Lowest</option>
                    </select>
                    <button
                        className="action-button"
                        onClick={() => setShowCreateForm(true)}
                    >
                        <Plus size={16} />
                        New Task
                    </button>
                </div>
            </div>

            <div className="page-content">
                {error && (
                    <div className="error-message">
                        {error}
                    </div>
                )}

                {isLoading && tasks.length === 0 ? (
                    <div className="loading-state">
                        <div className="spinner"></div>
                        <p>Loading tasks...</p>
                    </div>
                ) : tasks.length === 0 ? (
                    <div className="empty-state">
                        <div className="empty-icon">✅</div>
                        <h3>No Tasks Yet</h3>
                        <p>Add your first task and let Xarvis help you manage and prioritize your work.</p>
                        <button
                            className="cta-button"
                            onClick={() => setShowCreateForm(true)}
                        >
                            <Plus size={16} />
                            Add Task
                        </button>
                    </div>
                ) : (
                    <div className="tasks-table-container">
                        <table className="tasks-table">
                            <thead>
                                <tr>
                                    <th>
                                        <button
                                            className="sort-button"
                                            onClick={() => {
                                                if (sortBy === 'title') {
                                                    setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
                                                } else {
                                                    setSortBy('title');
                                                    setSortOrder('asc');
                                                }
                                            }}
                                        >
                                            Task {sortBy === 'title' && (sortOrder === 'asc' ? '↑' : '↓')}
                                        </button>
                                    </th>
                                    <th>Status</th>
                                    <th>
                                        <button
                                            className="sort-button"
                                            onClick={() => {
                                                if (sortBy === 'priority') {
                                                    setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
                                                } else {
                                                    setSortBy('priority');
                                                    setSortOrder('desc');
                                                }
                                            }}
                                        >
                                            Priority {sortBy === 'priority' && (sortOrder === 'asc' ? '↑' : '↓')}
                                        </button>
                                    </th>
                                    <th>
                                        <button
                                            className="sort-button"
                                            onClick={() => {
                                                if (sortBy === 'dueAt') {
                                                    setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
                                                } else {
                                                    setSortBy('dueAt');
                                                    setSortOrder('asc');
                                                }
                                            }}
                                        >
                                            Due Date {sortBy === 'dueAt' && (sortOrder === 'asc' ? '↑' : '↓')}
                                        </button>
                                    </th>
                                    <th>Tags</th>
                                    <th>
                                        <button
                                            className="sort-button"
                                            onClick={() => {
                                                if (sortBy === 'createdAt') {
                                                    setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
                                                } else {
                                                    setSortBy('createdAt');
                                                    setSortOrder('desc');
                                                }
                                            }}
                                        >
                                            Created {sortBy === 'createdAt' && (sortOrder === 'asc' ? '↑' : '↓')}
                                        </button>
                                    </th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {tasks.map((task) => (
                                    <tr key={task.id} className={isOverdue(task.dueAt) ? 'overdue' : ''}>
                                        <td className="task-title-cell">
                                            <div className="task-title">
                                                {task.title}
                                                {task.description && (
                                                    <div className="task-description">
                                                        {task.description.length > 100
                                                            ? `${task.description.substring(0, 100)}...`
                                                            : task.description
                                                        }
                                                    </div>
                                                )}
                                            </div>
                                        </td>
                                        <td>
                                            <div className="status-cell">
                                                {getStatusIcon(task.status)}
                                                <select
                                                    value={task.status}
                                                    onChange={(e) => handleStatusChange(task.id, e.target.value as TaskStatus)}
                                                    className="status-select"
                                                    style={{ color: getStatusColor(task.status) }}
                                                >
                                                    <option value="pending">Pending</option>
                                                    <option value="done">Done</option>
                                                    <option value="cancelled">Cancelled</option>
                                                </select>
                                            </div>
                                        </td>
                                        <td>
                                            <span
                                                className="priority-badge"
                                                style={{ backgroundColor: getPriorityColor(task.priority) }}
                                            >
                                                {getPriorityLabel(task.priority)}
                                            </span>
                                        </td>
                                        <td className={isOverdue(task.dueAt) ? 'overdue-date' : ''}>
                                            {task.dueAt ? (
                                                <div className="due-date">
                                                    <Calendar size={14} />
                                                    {formatDate(task.dueAt)}
                                                    {isOverdue(task.dueAt) && <AlertTriangle size={14} className="overdue-icon" />}
                                                </div>
                                            ) : (
                                                <span className="no-date">No due date</span>
                                            )}
                                        </td>
                                        <td>
                                            {task.tags && task.tags.length > 0 && (
                                                <div className="task-tags">
                                                    {task.tags.slice(0, 2).map((tag, index) => (
                                                        <span key={index} className="tag">
                                                            <Tag size={12} />
                                                            {tag}
                                                        </span>
                                                    ))}
                                                    {task.tags.length > 2 && (
                                                        <span className="tag more">+{task.tags.length - 2}</span>
                                                    )}
                                                </div>
                                            )}
                                        </td>
                                        <td className="created-date">
                                            {formatDate(task.createdAt)}
                                        </td>
                                        <td>
                                            <div className="task-actions">
                                                <button
                                                    onClick={() => setEditingTask(task)}
                                                    className="action-button small"
                                                    title="Edit task"
                                                >
                                                    <Edit size={14} />
                                                </button>
                                                <button
                                                    onClick={() => handleDeleteTask(task.id)}
                                                    className="action-button small danger"
                                                    title="Delete task"
                                                >
                                                    <Trash2 size={14} />
                                                </button>
                                            </div>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}

                {showCreateForm && (
                    <TaskForm
                        onSubmit={handleCreateTask}
                        onCancel={() => setShowCreateForm(false)}
                        isLoading={isLoading}
                    />
                )}

                {editingTask && (
                    <TaskForm
                        task={editingTask}
                        onSubmit={(data: UpdateTaskRequest) => handleUpdateTask(editingTask.id, data)}
                        onCancel={() => setEditingTask(null)}
                        isLoading={isLoading}
                    />
                )}
            </div>
        </div>
    );
};

interface TaskFormProps {
    task?: TaskResponse;
    onSubmit: (data: any) => void;
    onCancel: () => void;
    isLoading: boolean;
}

const TaskForm: React.FC<TaskFormProps> = ({ task, onSubmit, onCancel, isLoading }) => {
    const [title, setTitle] = useState(task?.title || '');
    const [description, setDescription] = useState(task?.description || '');
    const [priority, setPriority] = useState<TaskPriority>(task?.priority || 3);
    const [tags, setTags] = useState(task?.tags?.join(', ') || '');
    const [scheduledAt, setScheduledAt] = useState(task?.scheduledAt ? new Date(task.scheduledAt).toISOString().split('T')[0] : '');
    const [dueAt, setDueAt] = useState(task?.dueAt ? new Date(task.dueAt).toISOString().split('T')[0] : '');
    const [isRecurring, setIsRecurring] = useState(task?.isRecurring || false);

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        const tagArray = tags.split(',').map(t => t.trim()).filter(t => t);

        const taskData = {
            title,
            description: description || undefined,
            priority,
            tags: tagArray,
            scheduledAt: scheduledAt || undefined,
            dueAt: dueAt || undefined,
            isRecurring,
        };

        onSubmit(taskData);
    };

    return (
        <div className="modal-overlay">
            <div className="modal">
                <div className="modal-header">
                    <h3>{task ? 'Edit Task' : 'Create New Task'}</h3>
                </div>
                <form onSubmit={handleSubmit} className="modal-body">
                    <div className="form-group">
                        <label htmlFor="title">Title *</label>
                        <input
                            id="title"
                            type="text"
                            value={title}
                            onChange={(e) => setTitle(e.target.value)}
                            placeholder="Task title"
                            required
                        />
                    </div>

                    <div className="form-group">
                        <label htmlFor="description">Description</label>
                        <textarea
                            id="description"
                            value={description}
                            onChange={(e) => setDescription(e.target.value)}
                            placeholder="Task description"
                            rows={3}
                        />
                    </div>

                    <div className="form-row">
                        <div className="form-group">
                            <label htmlFor="priority">Priority</label>
                            <select
                                id="priority"
                                value={priority}
                                onChange={(e) => setPriority(parseInt(e.target.value) as TaskPriority)}
                            >
                                <option value="1">Lowest</option>
                                <option value="2">Low</option>
                                <option value="3">Medium</option>
                                <option value="4">High</option>
                                <option value="5">Urgent</option>
                            </select>
                        </div>

                        <div className="form-group">
                            <label htmlFor="isRecurring">Recurring</label>
                            <input
                                id="isRecurring"
                                type="checkbox"
                                checked={isRecurring}
                                onChange={(e) => setIsRecurring(e.target.checked)}
                            />
                        </div>
                    </div>

                    <div className="form-row">
                        <div className="form-group">
                            <label htmlFor="scheduledAt">Scheduled At</label>
                            <input
                                id="scheduledAt"
                                type="datetime-local"
                                value={scheduledAt}
                                onChange={(e) => setScheduledAt(e.target.value)}
                            />
                        </div>

                        <div className="form-group">
                            <label htmlFor="dueAt">Due Date</label>
                            <input
                                id="dueAt"
                                type="datetime-local"
                                value={dueAt}
                                onChange={(e) => setDueAt(e.target.value)}
                            />
                        </div>
                    </div>

                    <div className="form-group">
                        <label htmlFor="tags">Tags (comma-separated)</label>
                        <input
                            id="tags"
                            type="text"
                            value={tags}
                            onChange={(e) => setTags(e.target.value)}
                            placeholder="work, urgent, personal"
                        />
                    </div>

                    <div className="modal-actions">
                        <button type="button" onClick={onCancel} className="cancel-button">
                            Cancel
                        </button>
                        <button type="submit" disabled={isLoading || !title.trim()}>
                            {isLoading ? 'Saving...' : (task ? 'Update' : 'Create')}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export const ConnectionsPage: React.FC = () => {
    return (
        <div className="page-container">
            <div className="page-header">
                <div className="header-left">
                    <h1>Connections</h1>
                    <p>Manage your network and collaboration connections</p>
                </div>
                <div className="header-actions">
                    <button className="action-button secondary">
                        <Search size={16} />
                        Search
                    </button>
                    <button className="action-button">
                        <Plus size={16} />
                        Add Connection
                    </button>
                </div>
            </div>

            <div className="page-content">
                <div className="empty-state">
                    <div className="empty-icon">🤝</div>
                    <h3>No Connections Yet</h3>
                    <p>Start building your network and connect with others through Xarvis.</p>
                    <button className="cta-button">
                        <Plus size={16} />
                        Add Connection
                    </button>
                </div>
            </div>
        </div>
    );
};
