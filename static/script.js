// Simple upload handling for cataloger

// Model options for each provider
const modelOptions = {
    ollama: [
        'mistral-small3.2:24b',
        'llama3.2-vision:latest',
        'llava:latest'
    ],
    openai: [
        'gpt-4o',
        'gpt-4o-mini',
        'gpt-4-turbo'
    ]
};

function updateModelOptions() {
    const providerSelect = document.getElementById('provider-select');
    const modelSelect = document.getElementById('model-select');
    const provider = providerSelect.value;

    // Clear existing options
    modelSelect.innerHTML = '';

    // Add new options
    const models = modelOptions[provider] || [];
    models.forEach(model => {
        const option = document.createElement('option');
        option.value = model;
        option.textContent = model;
        modelSelect.appendChild(option);
    });
}

// Initialize model options on page load
document.addEventListener('DOMContentLoaded', () => {
    updateModelOptions();
    loadSessions();
});

async function handleUpload() {
    const fileInput = document.getElementById('file-input');
    const imageTypeSelect = document.getElementById('image-type-select');
    const providerSelect = document.getElementById('provider-select');
    const modelSelect = document.getElementById('model-select');
    const file = fileInput.files[0];

    if (!file) {
        alert('Please select a file');
        return;
    }

    const formData = new FormData();
    formData.append('file', file);
    formData.append('image_type', imageTypeSelect.value);
    formData.append('provider', providerSelect.value);
    formData.append('model', modelSelect.value);

    try {
        const response = await fetch('/api/upload', {
            method: 'POST',
            body: formData
        });

        if (!response.ok) {
            throw new Error(`Upload failed: ${response.statusText}`);
        }

        const result = await response.json();

        // Clear input
        fileInput.value = '';

        // Reload sessions list
        loadSessions();

        // Go directly to the session
        viewSession(result.session_id);
    } catch (error) {
        console.error('Upload error:', error);
        alert('Upload failed: ' + error.message);
    }
}

async function handleUrlUpload() {
    const urlInput = document.getElementById('url-input');
    const imageTypeSelect = document.getElementById('image-type-url-select');
    const providerSelect = document.getElementById('provider-select');
    const modelSelect = document.getElementById('model-select');
    const imageURL = urlInput.value.trim();

    if (!imageURL) {
        alert('Please enter an image URL');
        return;
    }

    try {
        const response = await fetch('/api/upload', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                image_url: imageURL,
                image_type: imageTypeSelect.value,
                provider: providerSelect.value,
                model: modelSelect.value
            })
        });

        if (!response.ok) {
            throw new Error(`Upload failed: ${response.statusText}`);
        }

        const result = await response.json();

        // Clear input
        urlInput.value = '';

        // Reload sessions list
        loadSessions();

        // Go directly to the session
        viewSession(result.session_id);
    } catch (error) {
        console.error('Upload error:', error);
        alert('Upload failed: ' + error.message);
    }
}

async function loadSessions() {
    try {
        const response = await fetch('/api/sessions');
        const sessions = await response.json();

        const sessionsList = document.getElementById('sessions-list');

        if (!sessions || sessions.length === 0) {
            sessionsList.innerHTML = '<p style="color: #888;">No sessions yet</p>';
            return;
        }

        sessionsList.innerHTML = sessions.map(session => `
            <div class="session-item" onclick="viewSession('${session.id}')" style="cursor: pointer;">
                <div class="session-info">
                    <strong>${session.id}</strong>
                    <br>
                    <small>Created: ${new Date(session.created_at).toLocaleString()}</small>
                    <br>
                    <small>Images: ${session.images?.length || 0}</small>
                    ${session.marc_record ? '<br><span style="color: #4ade80;">✓ MARC record available</span>' : ''}
                </div>
            </div>
        `).join('');
    } catch (error) {
        console.error('Failed to load sessions:', error);
        document.getElementById('sessions-list').innerHTML = '<p style="color: #ff6b6b;">Failed to load sessions</p>';
    }
}

async function viewSession(sessionId) {
    try {
        const response = await fetch(`/api/sessions/${sessionId}`);
        if (!response.ok) {
            throw new Error('Failed to fetch session');
        }

        const session = await response.json();

        // Show modal with session details
        showSessionModal(session);
    } catch (error) {
        console.error('Failed to load session:', error);
        alert('Failed to load session details');
    }
}

function showSessionModal(session) {
    // Create modal if it doesn't exist
    let modal = document.getElementById('session-modal');
    if (!modal) {
        modal = document.createElement('div');
        modal.id = 'session-modal';
        modal.className = 'modal';
        modal.innerHTML = `
            <div class="modal-content">
                <span class="close" onclick="closeSessionModal()">&times;</span>
                <div id="modal-body"></div>
            </div>
        `;
        document.body.appendChild(modal);
    }

    const modalBody = document.getElementById('modal-body');

    let content = `<h2>Session: ${session.id}</h2>`;
    content += `<p><small>Created: ${new Date(session.created_at).toLocaleString()}</small></p>`;
    if (session.provider) {
        content += `<p><small><strong>Provider:</strong> ${session.provider} | <strong>Model:</strong> ${session.model}</small></p>`;
    }

    // Show images
    if (session.images && session.images.length > 0) {
        content += `<h3>Images</h3>`;
        session.images.forEach(img => {
            content += `
                <div style="margin: 10px 0;">
                    <p><strong>Type:</strong> ${img.image_type}</p>
                    <img src="${img.image_url}" style="max-width: 400px; border: 1px solid #333; border-radius: 4px;" />
                </div>
            `;
        });
    }

    // Show MARC record
    if (session.marc_record) {
        content += `
            <h3>MARC Record</h3>
            <pre style="background: #1a1a1a; padding: 15px; border-radius: 4px; overflow-x: auto; white-space: pre-wrap; word-wrap: break-word;">${session.marc_record}</pre>
        `;
    }

    modalBody.innerHTML = content;
    modal.style.display = 'block';
}

function closeSessionModal() {
    const modal = document.getElementById('session-modal');
    if (modal) {
        modal.style.display = 'none';
    }
}

// Close modal when clicking outside
window.onclick = function(event) {
    const modal = document.getElementById('session-modal');
    if (event.target === modal) {
        closeSessionModal();
    }
};
