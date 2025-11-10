<script lang="ts">
import Modal from './Modal.svelte';
import { openModal, closeModal, closeAllModals } from '$core/store.svelte';

// Example modal IDs
const MODAL_SIMPLE = 1;
const MODAL_WITH_ACTIONS = 2;
const MODAL_CUSTOM_SIZE = 3;

function handleSave() {
  console.log('Save clicked');
  closeAllModals();
}

function handleDelete() {
  console.log('Delete clicked');
  closeAllModals();
}

function handleClose() {
  console.log('Close clicked');
}
</script>

<div class="example-container">
  <h1>Modal Component Examples</h1>

  <!-- Buttons to open modals -->
  <div class="button-group">
    <button onclick={() => openModal(MODAL_SIMPLE)}>
      Open Simple Modal
    </button>
    <button onclick={() => openModal(MODAL_WITH_ACTIONS)}>
      Open Modal with Actions
    </button>
    <button onclick={() => openModal(MODAL_CUSTOM_SIZE)}>
      Open Custom Size Modal
    </button>
  </div>

  <!-- Simple Modal -->
  <Modal id={MODAL_SIMPLE} title="Simple Modal" size={5}>
    <div class="modal-content">
      <p>This is a simple modal with just a title and close button.</p>
      <p>You can add any content here.</p>
    </div>
  </Modal>

  <!-- Modal with Save and Delete actions -->
  <Modal 
    id={MODAL_WITH_ACTIONS} 
    title="Modal with Actions"
    size={6}
    isEdit={true}
    onSave={handleSave}
    onDelete={handleDelete}
    onClose={handleClose}
  >
    <div class="modal-content">
      <h3>Edit Form</h3>
      <p>This modal has save and delete buttons.</p>
      <form>
        <div class="form-group">
          <label for="name">Name:</label>
          <input type="text" id="name" placeholder="Enter name" />
        </div>
        <div class="form-group">
          <label for="email">Email:</label>
          <input type="email" id="email" placeholder="Enter email" />
        </div>
      </form>
    </div>
  </Modal>

  <!-- Modal with custom size -->
  <Modal 
    id={MODAL_CUSTOM_SIZE} 
    title="Large Modal"
    size={9}
  >
    <div class="modal-content">
      <h3>Large Modal with Custom Size</h3>
      <p>This modal uses size 9 for the largest available size (1000px / 88vw).</p>
      <p>Available sizes (1-9):</p>
      <ul>
        <li><code>1</code> - 600px / 64vw</li>
        <li><code>5</code> - 800px / 75vw</li>
        <li><code>9</code> - 1000px / 88vw</li>
      </ul>
    </div>
  </Modal>
</div>

<style>
  .example-container {
    padding: 2rem;
  }

  .button-group {
    display: flex;
    gap: 1rem;
    margin-bottom: 2rem;
  }

  button {
    padding: 0.5rem 1rem;
    background: #0098f7;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }

  button:hover {
    background: #007acc;
  }

  .modal-content {
    padding: 1rem 0;
  }

  .form-group {
    margin-bottom: 1rem;
  }

  .form-group label {
    display: block;
    margin-bottom: 0.5rem;
    font-weight: 600;
  }

  .form-group input {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid #ccc;
    border-radius: 4px;
  }

  code {
    background: #f0f0f0;
    padding: 0.2rem 0.4rem;
    border-radius: 3px;
    font-family: monospace;
  }
</style>

