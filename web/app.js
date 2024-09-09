document.addEventListener('DOMContentLoaded', function() {
    const form = document.getElementById('menuForm');
    const submitButton = document.getElementById('submitButton');
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
        
        // Change button text and disable it
        submitButton.textContent = 'Processing...';
        submitButton.disabled = true;

        const formData = new FormData(form);
        const data = Object.fromEntries(formData);

        // Add date range to data
        const selectedDates = fpicker.selectedDates;
        data.startDate = formatDate(selectedDates[0]);
        data.endDate = selectedDates[1] ? formatDate(selectedDates[1]) : data.startDate;

        try {
            console.log('Sending request with data:', data);
            const response = await fetch('/api/generate', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(data),
            });

            console.log('Response status:', response.status);
            console.log('Response headers:', Object.fromEntries(response.headers.entries()));

            if (!response.ok) {
                const errorText = await response.text();
                console.error('Error response:', errorText);
                throw new Error(`HTTP error! status: ${response.status}, message: ${errorText}`);
            }

            if (data.action === 'ics') {
                const contentType = response.headers.get('Content-Type');
                console.log('Content-Type:', contentType);

                const text = await response.text();
                console.log('Response as text:', text);

                const blob = new Blob([text], {type: 'text/calendar'});
                console.log('Blob size:', blob.size);
                console.log('Blob type:', blob.type);

                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.style.display = 'none';
                a.href = url;
                a.download = `lunch_menu_${data.startDate}_to_${data.endDate}.ics`;
                document.body.appendChild(a);
                a.click();
                window.URL.revokeObjectURL(url);
                document.getElementById('result').textContent = 'Calendar file downloaded successfully.';
            } else {
                const result = await response.json();
                document.getElementById('result').textContent = result.message;
            }
        } catch (error) {
            console.error('Error:', error);
            document.getElementById('result').textContent = `An error occurred: ${error.message}`;
        } finally {
            // Reset button text and re-enable it
            submitButton.textContent = 'Submit';
            submitButton.disabled = false;
        }
    });

    function formatDate(date) {
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        const year = date.getFullYear();
        return `${month}-${day}-${year}`;
    }
});
