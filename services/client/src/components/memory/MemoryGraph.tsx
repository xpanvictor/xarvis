import React, { useCallback, useMemo, useState } from 'react';
import ReactFlow, {
    Node,
    Edge,
    addEdge,
    useNodesState,
    useEdgesState,
    MiniMap,
    Controls,
    Background,
    Connection,
    ReactFlowProvider,
    NodeChange,
    EdgeChange,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { Memory } from '../../services/api';
import { useConversationStore } from '../../store';
import './MemoryGraph.css';

interface MemoryNode extends Node {
    data: {
        memory: Memory;
        label: string;
        type: 'episodic' | 'semantic';
        saliency: number;
    };
}

interface MemoryGraphProps {
    className?: string;
}

// Custom node component for memories
const MemoryNodeComponent: React.FC<{ data: MemoryNode['data'] }> = ({ data }) => {
    const { memory, type, saliency } = data;
    const isEpisodic = type === 'episodic';

    // Calculate node size based on saliency score
    const nodeSize = Math.max(80, Math.min(120, 80 + (saliency / 10)));

    return (
        <div
            className={`memory-node ${type}`}
            style={{
                width: nodeSize,
                height: nodeSize,
            }}
        >
            <div className="memory-content">
                <div className="memory-type">{isEpisodic ? '📅' : '🧠'}</div>
                <div className="memory-text">
                    {memory.string.length > 50
                        ? memory.string.substring(0, 50) + '...'
                        : memory.string
                    }
                </div>
                <div className="memory-saliency">{saliency}</div>
            </div>

            {/* Pulsing effect for high saliency memories */}
            {saliency > 50 && <div className="saliency-pulse" />}
        </div>
    );
};

// Define custom node types
const nodeTypes = {
    memory: MemoryNodeComponent,
};

export const MemoryGraph: React.FC<MemoryGraphProps> = ({ className = '' }) => {
    const { memories } = useConversationStore();
    const [selectedMemory, setSelectedMemory] = useState<Memory | null>(null);

    // Convert memories to nodes
    const initialNodes: MemoryNode[] = useMemo(() => {
        return memories.map((memory, index) => {
            // Calculate position using a spiral layout
            const angle = (index * 137.5) * (Math.PI / 180); // Golden angle
            const radius = Math.sqrt(index) * 30 + 100;
            const x = Math.cos(angle) * radius + 400;
            const y = Math.sin(angle) * radius + 300;

            return {
                id: memory.id,
                type: 'memory',
                position: { x, y },
                data: {
                    memory,
                    label: memory.string.substring(0, 30) + (memory.string.length > 30 ? '...' : ''),
                    type: memory.memory_type,
                    saliency: memory.saliency_score,
                },
                draggable: true,
            };
        });
    }, [memories]);

    // Generate edges based on memory relationships
    const initialEdges: Edge[] = useMemo(() => {
        const edges: Edge[] = [];

        // Simple connection logic - connect memories with similar words or concepts
        for (let i = 0; i < memories.length; i++) {
            for (let j = i + 1; j < memories.length; j++) {
                const memory1 = memories[i];
                const memory2 = memories[j];

                // Basic similarity check (you could implement more sophisticated NLP here)
                const similarity = calculateSimilarity(memory1.string, memory2.string);

                if (similarity > 0.3) { // Threshold for connection
                    edges.push({
                        id: `${memory1.id}-${memory2.id}`,
                        source: memory1.id,
                        target: memory2.id,
                        animated: similarity > 0.6,
                        style: {
                            stroke: similarity > 0.6 ? '#64ffda' : '#888',
                            strokeWidth: Math.max(1, similarity * 3),
                        },
                        label: similarity > 0.8 ? 'Strong' : undefined,
                    });
                }
            }
        }

        return edges;
    }, [memories]);

    const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
    const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

    const onConnect = useCallback(
        (params: Connection) => setEdges((eds) => addEdge(params, eds)),
        [setEdges]
    );

    const onNodeClick = useCallback((event: React.MouseEvent, node: Node) => {
        const memoryNode = node as MemoryNode;
        setSelectedMemory(memoryNode.data.memory);
    }, []);

    // Update nodes when memories change
    React.useEffect(() => {
        setNodes(initialNodes);
    }, [initialNodes, setNodes]);

    React.useEffect(() => {
        setEdges(initialEdges);
    }, [initialEdges, setEdges]);

    if (memories.length === 0) {
        return (
            <div className={`memory-graph-empty ${className}`}>
                <div className="empty-state">
                    <div className="empty-icon">🧠</div>
                    <h3>No Memories Yet</h3>
                    <p>Start a conversation to build your memory graph</p>
                </div>
            </div>
        );
    }

    return (
        <div className={`memory-graph-container ${className}`}>
            <ReactFlowProvider>
                <ReactFlow
                    nodes={nodes}
                    edges={edges}
                    onNodesChange={onNodesChange}
                    onEdgesChange={onEdgesChange}
                    onConnect={onConnect}
                    onNodeClick={onNodeClick}
                    nodeTypes={nodeTypes}
                    fitView
                    fitViewOptions={{ padding: 0.2 }}
                    minZoom={0.2}
                    maxZoom={2}
                    className="memory-graph"
                >
                    <Background />
                    <Controls />
                    <MiniMap
                        nodeColor="#64ffda"
                        maskColor="rgba(0, 0, 0, 0.8)"
                        className="memory-minimap"
                    />
                </ReactFlow>
            </ReactFlowProvider>

            {/* Memory Details Panel */}
            {selectedMemory && (
                <div className="memory-details-panel">
                    <div className="memory-details-header">
                        <h3>Memory Details</h3>
                        <button
                            className="close-panel"
                            onClick={() => setSelectedMemory(null)}
                        >
                            ✕
                        </button>
                    </div>

                    <div className="memory-details-content">
                        <div className="memory-detail-item">
                            <label>Type:</label>
                            <span className={`memory-type-badge ${selectedMemory.memory_type}`}>
                                {selectedMemory.memory_type === 'episodic' ? '📅 Episodic' : '🧠 Semantic'}
                            </span>
                        </div>

                        <div className="memory-detail-item">
                            <label>Content:</label>
                            <p>{selectedMemory.string}</p>
                        </div>

                        <div className="memory-detail-item">
                            <label>Saliency Score:</label>
                            <div className="saliency-bar">
                                <div
                                    className="saliency-fill"
                                    style={{ width: `${Math.min(100, selectedMemory.saliency_score)}%` }}
                                />
                                <span>{selectedMemory.saliency_score}</span>
                            </div>
                        </div>

                        <div className="memory-detail-item">
                            <label>Created:</label>
                            <span>{new Date(selectedMemory.created_at).toLocaleString()}</span>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

// Simple text similarity function (could be enhanced with proper NLP)
function calculateSimilarity(text1: string, text2: string): number {
    const words1 = text1.toLowerCase().split(/\W+/);
    const words2 = text2.toLowerCase().split(/\W+/);

    const intersection = words1.filter(word => words2.includes(word));
    const union = [...new Set([...words1, ...words2])];

    return intersection.length / union.length;
}

export default MemoryGraph;
