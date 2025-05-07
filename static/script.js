document.addEventListener('DOMContentLoaded', function() {
    // Cache DOM elements
    const addWebsiteForm = document.getElementById('addWebsiteForm');
    const websiteUrl = document.getElementById('websiteUrl');
    const websiteName = document.getElementById('websiteName');
    const usePKI = document.getElementById('usePKI');
    const pkiOptionsDiv = document.querySelector('.pki-options');
    const clientCertPath = document.getElementById('clientCertPath');
    const clientKeyPath = document.getElementById('clientKeyPath');
    const customRootCAPath = document.getElementById('customRootCAPath');
    const skipTLSVerify = document.getElementById('skipTLSVerify');
    const changedWebsitesList = document.getElementById('changedWebsitesList');
    const unchangedWebsitesList = document.getElementById('unchangedWebsitesList');
    const checkAllBtn = document.getElementById('checkAllBtn');
    const websiteItemTemplate = document.getElementById('websiteItemTemplate');
    
    // Load websites on page load
    loadWebsites();
    
    // Set up interval to refresh website data
    setInterval(loadWebsites, 60000); // Refresh every minute
    
    // Event listeners
    addWebsiteForm.addEventListener('submit', handleAddWebsite);
    checkAllBtn.addEventListener('click', handleCheckAll);
    
    // Show/hide PKI options based on checkbox
    usePKI.addEventListener('change', function() {
        if (this.checked) {
            pkiOptionsDiv.style.display = 'block';
        } else {
            pkiOptionsDiv.style.display = 'none';
        }
    });
    
    // Functions
    async function loadWebsites() {
        try {
            const response = await fetch('/api/websites');
            
            if (!response.ok) {
                throw new Error(`HTTP error! Status: ${response.status}`);
            }
            
            const websites = await response.json();
            renderWebsites(websites);
        } catch (error) {
            console.error('Error loading websites:', error);
            showError('Failed to load websites. Please try again later.');
        }
    }
    
    function renderWebsites(websites) {
        // Clear current lists
        changedWebsitesList.innerHTML = '';
        unchangedWebsitesList.innerHTML = '';
        
        // Check if there are any websites
        if (websites.length === 0) {
            changedWebsitesList.innerHTML = '<div class="empty-message">No websites being monitored</div>';
            unchangedWebsitesList.innerHTML = '<div class="empty-message">No websites being monitored</div>';
            return;
        }
        
        // Count for each category
        let changedCount = 0;
        let unchangedCount = 0;
        
        // Render each website
        websites.forEach(website => {
            const itemClone = document.importNode(websiteItemTemplate.content, true);
            
            // Fill in website details
            itemClone.querySelector('.website-name').textContent = website.name;
            itemClone.querySelector('.website-url').textContent = website.url;
            
            const lastCheckedSpan = itemClone.querySelector('.website-last-checked span');
            if (website.lastChecked && new Date(website.lastChecked).getTime() > 0) {
                lastCheckedSpan.textContent = formatDate(new Date(website.lastChecked));
            } else {
                lastCheckedSpan.textContent = 'Not yet checked';
            }
            
            const statusElement = itemClone.querySelector('.website-status');
            
            // Set the appropriate status
            if (website.error) {
                statusElement.textContent = `Error: ${website.error}`;
                statusElement.classList.add('error');
            } else if (website.isFirstCheck) {
                statusElement.textContent = 'Pending first check';
            } else if (website.hasChanged) {
                statusElement.textContent = 'Changed since last check';
                statusElement.classList.add('changed');
            } else {
                statusElement.textContent = 'No changes detected';
                statusElement.classList.add('unchanged');
            }
            
            // Set up button event listeners
            const websiteItem = itemClone.querySelector('.website-item');
            websiteItem.dataset.id = website.id;
            
            const checkNowBtn = itemClone.querySelector('.check-now-btn');
            checkNowBtn.addEventListener('click', () => handleCheckWebsite(website.id));
            
            const visitBtn = itemClone.querySelector('.visit-btn');
            visitBtn.addEventListener('click', () => window.open(website.url, '_blank'));
            
            const removeBtn = itemClone.querySelector('.remove-btn');
            removeBtn.addEventListener('click', () => handleRemoveWebsite(website.id));
            
            // Add to the appropriate list
            if (website.error || website.hasChanged) {
                changedWebsitesList.appendChild(itemClone);
                changedCount++;
            } else {
                unchangedWebsitesList.appendChild(itemClone);
                unchangedCount++;
            }
        });
        
        // Show empty messages if needed
        if (changedCount === 0) {
            changedWebsitesList.innerHTML = '<div class="empty-message">No changed websites found</div>';
        }
        
        if (unchangedCount === 0) {
            unchangedWebsitesList.innerHTML = '<div class="empty-message">No unchanged websites found</div>';
        }
    }
    
    async function handleAddWebsite(event) {
        event.preventDefault();
        
        const url = websiteUrl.value.trim();
        let name = websiteName.value.trim();
        
        // Validate URL
        if (!url) {
            showError('Please enter a valid URL');
            return;
        }
        
        // Use URL as name if name is not provided
        if (!name) {
            name = url;
        }
        
        // Prepare request data
        const requestData = {
            url,
            name,
            usePKI: usePKI.checked
        };
        
        // Add PKI fields if PKI is enabled
        if (usePKI.checked) {
            requestData.clientCertPath = clientCertPath.value.trim();
            requestData.clientKeyPath = clientKeyPath.value.trim();
            requestData.customRootCAPath = customRootCAPath.value.trim();
            requestData.skipTLSVerify = skipTLSVerify.checked;
            
            // Validate required certificate fields if they're provided
            if ((requestData.clientCertPath && !requestData.clientKeyPath) || 
                (!requestData.clientCertPath && requestData.clientKeyPath)) {
                showError('Both client certificate and key must be provided together');
                return;
            }
        }
        
        try {
            const response = await fetch('/api/websites', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(requestData)
            });
            
            if (!response.ok) {
                throw new Error(`HTTP error! Status: ${response.status}`);
            }
            
            // Reset form
            addWebsiteForm.reset();
            pkiOptionsDiv.style.display = 'none';
            
            // Reload websites
            loadWebsites();
        } catch (error) {
            console.error('Error adding website:', error);
            showError('Failed to add website. Please try again.');
        }
    }
    
    async function handleRemoveWebsite(id) {
        if (!confirm('Are you sure you want to remove this website from monitoring?')) {
            return;
        }
        
        try {
            const response = await fetch(`/api/websites/${id}`, {
                method: 'DELETE'
            });
            
            if (!response.ok) {
                throw new Error(`HTTP error! Status: ${response.status}`);
            }
            
            // Reload websites
            loadWebsites();
        } catch (error) {
            console.error('Error removing website:', error);
            showError('Failed to remove website. Please try again.');
        }
    }
    
    async function handleCheckWebsite(id) {
        try {
            const response = await fetch(`/api/websites/${id}/check`, {
                method: 'POST'
            });
            
            if (!response.ok) {
                throw new Error(`HTTP error! Status: ${response.status}`);
            }
            
            // Reload websites
            loadWebsites();
        } catch (error) {
            console.error('Error checking website:', error);
            showError('Failed to check website. Please try again.');
        }
    }
    
    async function handleCheckAll() {
        checkAllBtn.disabled = true;
        checkAllBtn.textContent = 'Checking...';
        
        try {
            const websitesResponse = await fetch('/api/websites');
            
            if (!websitesResponse.ok) {
                throw new Error(`HTTP error! Status: ${websitesResponse.status}`);
            }
            
            const websites = await websitesResponse.json();
            
            // Check each website
            const checkPromises = websites.map(website => 
                fetch(`/api/websites/${website.id}/check`, { method: 'POST' })
            );
            
            await Promise.all(checkPromises);
            
            // Reload websites
            loadWebsites();
        } catch (error) {
            console.error('Error checking all websites:', error);
            showError('Failed to check all websites. Please try again.');
        } finally {
            checkAllBtn.disabled = false;
            checkAllBtn.textContent = 'Check All Now';
        }
    }
    
    // Helper functions
    function formatDate(date) {
        const now = new Date();
        const diffInMinutes = Math.floor((now - date) / (1000 * 60));
        
        if (diffInMinutes < 1) {
            return 'Just now';
        } else if (diffInMinutes < 60) {
            return `${diffInMinutes} minute${diffInMinutes !== 1 ? 's' : ''} ago`;
        } else if (diffInMinutes < 24 * 60) {
            const hours = Math.floor(diffInMinutes / 60);
            return `${hours} hour${hours !== 1 ? 's' : ''} ago`;
        } else {
            const options = { 
                month: 'short', 
                day: 'numeric', 
                hour: '2-digit', 
                minute: '2-digit' 
            };
            return date.toLocaleDateString(undefined, options);
        }
    }
    
    function showError(message) {
        alert(message);
    }
});
