import React, { useState, useEffect } from 'react';
import './App.css'; // Asegúrate de tener este CSS o adapta el tuyo

// Configura la URL base de tu API Go
// Cuando esté en Docker Compose, 'backend' es el nombre del servicio Go.
// Para acceder desde el navegador local, usaremos localhost:3000 (el puerto mapeado del backend).
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:3000';

function App() {
  const [transactions, setTransactions] = useState([]);
  const [description, setDescription] = useState('');
  const [amount, setAmount] = useState('');
  const [type, setType] = useState('expense'); // 'expense' o 'income'
  const [editId, setEditId] = useState(null); // ID de la transacción en edición

  useEffect(() => {
    fetchTransactions();
  }, []);

  const fetchTransactions = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/transactions`);
      if (!response.ok) {
        throw new Error('No se pudieron obtener las transacciones');
      }
      const data = await response.json();
      setTransactions(data);
    } catch (error) {
      console.error('Error al obtener transacciones:', error);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!description || !amount || parseFloat(amount) <= 0) {
      alert('Por favor, introduce una descripción y un monto válido.');
      return;
    }

    const transaction = {
      description,
      amount: parseFloat(amount),
      type,
    };

    try {
      let response;
      if (editId) {
        // Actualizar transacción existente
        response = await fetch(`${API_BASE_URL}/transaction/${editId}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(transaction),
        });
        setEditId(null); // Sale del modo edición
      } else {
        // Crear nueva transacción
        response = await fetch(`${API_BASE_URL}/transaction`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(transaction),
        });
      }

      if (!response.ok) {
        throw new Error(`Error en la petición: ${response.statusText}`);
      }

      setDescription('');
      setAmount('');
      setType('expense');
      fetchTransactions(); // Refresca la lista
    } catch (error) {
      console.error('Error al guardar transacción:', error);
      alert(`Error: ${error.message}`);
    }
  };

  const handleEdit = (transaction) => {
    setDescription(transaction.description);
    setAmount(transaction.amount.toString());
    setType(transaction.type);
    setEditId(transaction.id);
  };

  const handleDelete = async (id) => {
    if (
      !window.confirm('¿Estás seguro de que quieres eliminar esta transacción?')
    ) {
      return;
    }
    try {
      const response = await fetch(`${API_BASE_URL}/transaction/${id}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        throw new Error('No se pudo eliminar la transacción');
      }
      fetchTransactions(); // Refresca la lista
    } catch (error) {
      console.error('Error al eliminar transacción:', error);
      alert(`Error: ${error.message}`);
    }
  };

  return (
    <div className="App">
      <h1>Registro de Transacciones</h1>

      <form onSubmit={handleSubmit}>
        <input
          type="text"
          placeholder="Descripción"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          required
        />
        <input
          type="number"
          placeholder="Monto"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          required
          min="0.01"
          step="0.01"
        />
        <select value={type} onChange={(e) => setType(e.target.value)}>
          <option value="expense">Gasto</option>
          <option value="income">Ingreso</option>
        </select>
        <button type="submit">
          {editId ? 'Actualizar Transacción' : 'Añadir Transacción'}
        </button>
        {editId && (
          <button type="button" onClick={() => setEditId(null)}>
            Cancelar Edición
          </button>
        )}
      </form>

      <h2>Historial de Transacciones</h2>
      {transactions.length === 0 ? (
        <p>No hay transacciones registradas.</p>
      ) : (
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>Descripción</th>
              <th>Monto</th>
              <th>Tipo</th>
              <th>Fecha</th>
              <th>Acciones</th>
            </tr>
          </thead>
          <tbody>
            {transactions.map((t) => (
              <tr key={t.id} className={t.type}>
                <td>{t.id}</td>
                <td>{t.description}</td>
                <td>{t.amount.toFixed(2)}</td>
                <td>{t.type === 'income' ? 'Ingreso' : 'Gasto'}</td>
                <td>{new Date(t.created_at).toLocaleDateString()}</td>
                <td>
                  <button onClick={() => handleEdit(t)}>Editar</button>
                  <button onClick={() => handleDelete(t.id)}>Eliminar</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

export default App;
