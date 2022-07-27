import "./EventForm.css";
export default function AbiEventForm(props) {
  const event = props.event;
  return (
    <div>
      <dl>
        <dt>Address</dt>
        <dd>{event.address}</dd>
        <dt>Name</dt>
        <dd>{event.name}</dd>
        <dt>Topics</dt>
        <dd>
          <ol start="0">
            <li>
              <strong>{event.topics[0]}</strong>
            </li>
            {event.topics.slice(1).map((topic) => {
              return <li>{topic}</li>;
            })}
          </ol>
        </dd>
        <dt>Data</dt>
        <dd>
          <div
            style={{
              width: "600px",
              textAlign: "justify",
              wordBreak: "break-all",
            }}
          >
            {event.data}
          </div>
        </dd>
      </dl>
    </div>
  );
}
