document.addEventListener('DOMContentLoaded', function() {
    const form = document.getElementById('menuForm');
    const actionSelect = document.getElementById('action');
    const emailFields = document.getElementById('emailFields');
    const dateRangeSelect = document.getElementById('dateRange');
    const datePicker = document.getElementById('datePicker');

    // Initialize Flatpickr
    let fpicker = flatpickr(datePicker, {
        mode: "single",
        dateFormat: "Y-m-d"
    });

    // Function to toggle email fields visibility
    function toggleEmailFields() {
        emailFields.style.display = actionSelect.value === 'email' ? 'block' : 'none';
    }

    // Initialize email fields visibility
    toggleEmailFields();

    // Update date picker based on date range selection
    dateRangeSelect.addEventListener('change', function() {
        switch(this.value) {
            case 'day':
                fpicker.set('mode', 'single');
                break;
            case 'week':
                fpicker.set('mode', 'range');
                break;
            case 'month':
                fpicker.set('mode', 'range');
                break;
        }
        fpicker.clear();
    });

    actionSelect.addEventListener('change', toggleEmailFields);

    form.addEventListener('submit', async function(e) {
        e.preventDefault();
        const formData = new FormData(form);
        const data = Object.fromEntries(formData);

        // Add date range to data
        const selectedDates = fpicker.selectedDates;
        data.startDate = formatDate(selectedDates[0]);
        data.endDate = selectedDates[1] ? formatDate(selectedDates[1]) : data.startDate;

        // Include recipients only if action is email
        if (data.action === 'email' && !data.recipients) {
            document.getElementById('result').textContent = 'Please enter recipient email address(es)';
            return;
        }

        try {
            const response = await fetch('/api/generate', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(data),
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            document.getElementById('result').textContent = result.message;
        } catch (error) {
            console.error('Error:', error);
            document.getElementById('result').textContent = 'An error occurred. Please try again.';
        }
    });
});

function formatDate(date) {
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const year = date.getFullYear();
    return `${month}-${day}-${year}`;
}
